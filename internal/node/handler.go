package node

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/snapek/claw-mesh/internal/types"
)

const maxNodeRequestBody = 1 << 20 // 1 MB

// Handler serves the node-side HTTP API for receiving forwarded messages.
type Handler struct {
	token string
	mux   *http.ServeMux
}

// NewHandler creates a node message handler.
// If token is non-empty, all requests must carry a matching Bearer token.
func NewHandler(token string) *Handler {
	h := &Handler{
		token: token,
		mux:   http.NewServeMux(),
	}
	h.mux.HandleFunc("POST /api/v1/messages", h.requireAuth(h.handleMessage))
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// requireAuth enforces Bearer token auth on the node handler.
func (h *Handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.token == "" {
			next(w, r)
			return
		}
		const prefix = "Bearer "
		auth := r.Header.Get("Authorization")
		if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix {
			writeNodeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing or invalid authorization header"})
			return
		}
		if auth[len(prefix):] != h.token {
			writeNodeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			return
		}
		next(w, r)
	}
}

// handleMessage receives a forwarded message from the coordinator.
// For now it echoes back the content as a placeholder for OpenClaw gateway integration.
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

	// Echo response — placeholder for actual OpenClaw gateway forwarding.
	resp := types.MessageResponse{
		MessageID: msg.ID,
		NodeID:    "", // filled by caller if needed
		Response:  msg.Content,
	}
	writeNodeJSON(w, http.StatusOK, resp)
}

func writeNodeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
