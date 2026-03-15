package node

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

const maxNodeRequestBody = 1 << 20 // 1 MB

// nodeTask tracks an in-progress async message on the node side.
type nodeTask struct {
	mu              sync.RWMutex
	MessageID       string
	Status          string // accepted, processing, completed, failed
	Response        string
	PartialResponse string
	Error           string
}

// Handler serves the node-side HTTP API for receiving forwarded messages.
type Handler struct {
	token         *string
	gatewayClient GatewayClient
	mux           *http.ServeMux

	tasksMu sync.RWMutex
	tasks   map[string]*nodeTask
}

// NewHandler creates a node message handler.
// If token is non-empty, all requests must carry a matching Bearer token.
// If gw is nil, messages are echoed back as a fallback.
func NewHandler(token *string, gw GatewayClient) *Handler {
	h := &Handler{
		token:         token,
		gatewayClient: gw,
		mux:           http.NewServeMux(),
		tasks:         make(map[string]*nodeTask),
	}
	h.mux.HandleFunc("POST /api/v1/messages", h.requireAuth(h.handleMessage))
	h.mux.HandleFunc("POST /api/v1/messages/async", h.requireAuth(h.handleMessageAsync))
	h.mux.HandleFunc("GET /api/v1/messages/{id}/status", h.requireAuth(h.handleMessageStatus))
	h.mux.HandleFunc("GET /healthz", h.handleHealthz)
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// requireAuth enforces Bearer token auth on the node handler.
func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.token == nil || *h.token == "" {
			next(w, r)
			return
		}
		const prefix = "Bearer "
		auth := r.Header.Get("Authorization")
		if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix {
			writeNodeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid authorization header"})
			return
		}
		if auth[len(prefix):] != *h.token {
			writeNodeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		next(w, r)
	}
}

// handleMessage receives a forwarded message from the coordinator.
// If a gateway client is configured, the message is forwarded to the local
// OpenClaw Gateway. Otherwise it echoes back as a fallback.
func (h *Handler) handleMessage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxNodeRequestBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var msg types.Message
	if err := dec.Decode(&msg); err != nil {
		writeNodeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid message body: %v", err)})
		return
	}

	if msg.ID == "" || msg.Content == "" {
		writeNodeJSON(w, http.StatusBadRequest, map[string]string{"error": "id and content are required"})
		return
	}

	log.Printf("received message %s: %s", msg.ID, msg.Content)

	if h.gatewayClient == nil {
		// Echo fallback — no gateway configured.
		log.Printf("WARN: no gateway client configured, echoing message %s", msg.ID)
		resp := types.MessageResponse{
			MessageID: msg.ID,
			NodeID:    "",
			Response:  "[claw-mesh] Gateway not available. Message: " + msg.Content,
		}
		writeNodeJSON(w, http.StatusOK, resp)
		return
	}

	// Forward to OpenClaw Gateway; fall back to echo on failure.
	gwResp, err := h.gatewayClient.SendMessage(r.Context(), &msg)
	if err != nil {
		log.Printf("gateway forwarding failed for message %s, falling back to echo: %v", msg.ID, err)
		resp := types.MessageResponse{
			MessageID: msg.ID,
			NodeID:    "",
			Response:  "[claw-mesh] Gateway error (echo fallback). Message: " + msg.Content,
		}
		writeNodeJSON(w, http.StatusOK, resp)
		return
	}

	writeNodeJSON(w, http.StatusOK, gwResp)
}

// handleMessageAsync accepts a message for async processing.
// Returns immediately with accepted status; processes in background.
func (h *Handler) handleMessageAsync(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxNodeRequestBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var msg types.Message
	if err := dec.Decode(&msg); err != nil {
		writeNodeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid message body: %v", err)})
		return
	}

	if msg.ID == "" || msg.Content == "" {
		writeNodeJSON(w, http.StatusBadRequest, map[string]string{"error": "id and content are required"})
		return
	}

	log.Printf("accepted async message %s: %s", msg.ID, msg.Content)

	nt := &nodeTask{
		MessageID: msg.ID,
		Status:    "accepted",
	}
	h.tasksMu.Lock()
	h.tasks[msg.ID] = nt
	h.tasksMu.Unlock()

	// Process in background.
	go h.processAsync(nt, &msg)

	writeNodeJSON(w, http.StatusAccepted, types.NodeAsyncAccepted{
		MessageID: msg.ID,
		Status:    "accepted",
	})
}

func (h *Handler) processAsync(nt *nodeTask, msg *types.Message) {
	nt.mu.Lock()
	nt.Status = "processing"
	nt.mu.Unlock()

	if h.gatewayClient == nil {
		log.Printf("WARN: no gateway client configured, echoing async message %s", msg.ID)
		nt.mu.Lock()
		nt.Status = "completed"
		nt.Response = "[claw-mesh] Gateway not available. Message: " + msg.Content
		nt.mu.Unlock()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	gwResp, err := h.gatewayClient.SendMessage(ctx, msg)
	nt.mu.Lock()
	defer nt.mu.Unlock()
	if err != nil {
		log.Printf("gateway forwarding failed for async message %s: %v", msg.ID, err)
		nt.Status = "failed"
		nt.Error = err.Error()
		return
	}
	nt.Status = "completed"
	nt.Response = gwResp.Response
}

// handleMessageStatus returns the current status of an async message.
func (h *Handler) handleMessageStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	h.tasksMu.RLock()
	nt := h.tasks[id]
	h.tasksMu.RUnlock()

	if nt == nil {
		writeNodeJSON(w, http.StatusNotFound, map[string]string{"error": "message not found"})
		return
	}

	nt.mu.RLock()
	resp := types.NodeMessageStatus{
		MessageID:       nt.MessageID,
		Status:          nt.Status,
		Response:        nt.Response,
		PartialResponse: nt.PartialResponse,
		Error:           nt.Error,
	}
	nt.mu.RUnlock()

	writeNodeJSON(w, http.StatusOK, resp)
}

// handleHealthz responds to active health probes from the coordinator.
func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeNodeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeNodeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
