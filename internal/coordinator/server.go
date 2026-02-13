package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/snapek/claw-mesh/internal/config"
	"github.com/snapek/claw-mesh/internal/types"
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
	rt := NewRouter(reg)
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
