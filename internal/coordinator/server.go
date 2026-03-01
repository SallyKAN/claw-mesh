package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/config"
	"github.com/SallyKAN/claw-mesh/internal/types"
)

const maxRequestBody = 1 << 20 // 1 MB

// Server is the coordinator HTTP server.
type Server struct {
	cfg       *config.CoordinatorConfig
	registry  *Registry
	router    *Router
	health    *HealthChecker
	forwarder *Forwarder
	http      *http.Server
}

// NewServer creates a coordinator server.
func NewServer(cfg *config.CoordinatorConfig) *Server {
	reg := NewRegistry()

	// Set up persistent store for routing rules.
	var store *Store
	dataDir := cfg.DataDir
	if dataDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			dataDir = filepath.Join(home, ".claw-mesh")
		} else {
			dataDir = "."
		}
	}
	storePath := filepath.Join(dataDir, "rules.json")
	if s, err := NewStore(storePath); err == nil {
		store = s
		log.Printf("rule store: %s", storePath)
	} else {
		log.Printf("WARN: could not init rule store at %s: %v", storePath, err)
	}

	rt := NewRouter(reg, store)
	hc := NewHealthChecker(reg, 30*time.Second, 10*time.Second)
	fwd := NewForwarder()

	s := &Server{
		cfg:       cfg,
		registry:  reg,
		router:    rt,
		health:    hc,
		forwarder: fwd,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/nodes/register", s.requireAuth(s.handleRegister))
	mux.HandleFunc("DELETE /api/v1/nodes/{id}", s.requireAuth(s.handleDeregister))
	mux.HandleFunc("GET /api/v1/nodes", s.handleListNodes)
	mux.HandleFunc("GET /api/v1/nodes/{id}", s.handleGetNode)
	mux.HandleFunc("POST /api/v1/nodes/{id}/heartbeat", s.requireAuth(s.handleHeartbeat))

	// Routing
	mux.HandleFunc("POST /api/v1/route", s.requireAuth(s.handleRouteAuto))
	mux.HandleFunc("POST /api/v1/route/{nodeId}", s.requireAuth(s.handleRouteToNode))
	mux.HandleFunc("GET /api/v1/rules", s.handleListRules)
	mux.HandleFunc("POST /api/v1/rules", s.requireAuth(s.handleAddRule))
	mux.HandleFunc("DELETE /api/v1/rules/{id}", s.requireAuth(s.handleDeleteRule))

	// Dashboard
	mux.Handle("/", DashboardHandler(cfg.Token))

	s.http = &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           recoverMiddleware(requestLogger(mux)),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return s
}

// Start begins serving and the health checker. Blocks until the server stops.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.http.Addr)
	if err != nil {
		return err
	}
	s.health.Start()
	log.Printf("coordinator listening on %s", s.http.Addr)
	return s.http.Serve(ln)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.health.Stop()
	return s.http.Shutdown(ctx)
}

// requireAuth wraps a handler to enforce Bearer token auth on mutating endpoints.
// Accepts the coordinator admin token or any valid per-node token.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Token == "" {
			next(w, r)
			return
		}
		const prefix = "Bearer "
		auth := r.Header.Get("Authorization")
		if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid authorization header"})
			return
		}
		token := auth[len(prefix):]
		if token != s.cfg.Token && !s.registry.ValidateNodeToken(token) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		next(w, r)
	}
}

// decodeJSON reads a JSON body with size limit and strict field checking.
// It rejects requests with trailing data after the JSON value.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	// Reject trailing garbage.
	if dec.More() {
		return fmt.Errorf("unexpected data after JSON value")
	}
	return nil
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req types.RegisterRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" || req.Endpoint == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and endpoint are required"})
		return
	}

	if err := validateEndpoint(req.Endpoint, s.cfg.AllowPrivate); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	id, err := generateUniqueID(s.registry.Exists)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate node ID"})
		return
	}

	nodeToken, err := generateToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate node token"})
		return
	}

	node := &types.Node{
		ID:            id,
		Name:          req.Name,
		Endpoint:      req.Endpoint,
		Capabilities:  req.Capabilities,
		Status:        types.NodeStatusOnline,
		LastHeartbeat: time.Now(),
	}

	if err := s.registry.Add(node); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	s.registry.SetNodeToken(node.ID, nodeToken)

	log.Printf("node registered: %s (%s) at %s", node.ID, node.Name, node.Endpoint)
	writeJSON(w, http.StatusCreated, types.RegisterResponse{
		NodeID: node.ID,
		Token:  nodeToken,
	})
}

func (s *Server) handleDeregister(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.registry.Remove(id) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
		return
	}
	log.Printf("node deregistered: %s", id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.registry.List())
}

func (s *Server) handleGetNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	node := s.registry.Get(id)
	if node == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
		return
	}
	writeJSON(w, http.StatusOK, node)
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req types.HeartbeatRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if !types.ValidNodeStatus(req.Status) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status value"})
		return
	}

	if !s.registry.RecordHeartbeat(id, req.Status) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "node not found"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// recoverMiddleware catches panics and returns 500 instead of crashing.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				log.Printf("panic: %v", rv)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// requestLogger logs method, path, and duration for each request.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// validateEndpoint checks that an endpoint is a valid host:port and rejects
// URLs with schemes/paths. Unless allowPrivate is true, loopback and private
// IPs are rejected to prevent SSRF.
func validateEndpoint(endpoint string, allowPrivate bool) error {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("endpoint must be host:port format: %v", err)
	}
	if host == "" || port == "" {
		return fmt.Errorf("endpoint must have both host and port")
	}

	// Reject anything that looks like a URL (contains / or scheme).
	for _, ch := range endpoint {
		if ch == '/' {
			return fmt.Errorf("endpoint must be host:port, not a URL")
		}
	}

	if allowPrivate {
		return nil
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Could be a hostname — resolve it.
		addrs, err := net.LookupHost(host)
		if err != nil {
			// Can't resolve — allow it (might be reachable from coordinator).
			return nil
		}
		for _, addr := range addrs {
			if parsed := net.ParseIP(addr); parsed != nil && isPrivateIP(parsed) {
				return fmt.Errorf("endpoint resolves to private/loopback IP %s (set allow_private to permit)", addr)
			}
		}
		return nil
	}

	if isPrivateIP(ip) {
		return fmt.Errorf("private/loopback endpoints not allowed (set allow_private to permit)")
	}
	return nil
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	privateRanges := []struct{ start, end net.IP }{
		{net.ParseIP("10.0.0.0"), net.ParseIP("10.255.255.255")},
		{net.ParseIP("172.16.0.0"), net.ParseIP("172.31.255.255")},
		{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.255.255")},
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	for _, r := range privateRanges {
		s := r.start.To4()
		e := r.end.To4()
		if bytesInRange(ip4, s, e) {
			return true
		}
	}
	return false
}

func bytesInRange(ip, start, end net.IP) bool {
	for i := 0; i < len(ip); i++ {
		if ip[i] < start[i] {
			return false
		}
		if ip[i] > end[i] {
			return false
		}
	}
	return true
}
