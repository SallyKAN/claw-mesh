package node

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

const maxNodeRequestBody = 1 << 20 // 1 MB

// Handler serves the node-side HTTP API for receiving forwarded messages.
type Handler struct {
	token         *string
	gatewayClient GatewayClient
	mux           *http.ServeMux
}

// NewHandler creates a node message handler.
// If token is non-empty, all requests must carry a matching Bearer token.
// If gw is nil, messages are echoed back as a fallback.
func NewHandler(token *string, gw GatewayClient) *Handler {
	h := &Handler{
		token:         token,
		gatewayClient: gw,
		mux:           http.NewServeMux(),
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

	// Forward to OpenClaw Gateway.
	gwResp, err := h.gatewayClient.SendMessage(r.Context(), &msg)
	if err != nil {
		log.Printf("gateway forwarding failed for message %s: %v", msg.ID, err)
		writeNodeJSON(w, http.StatusBadGateway, map[string]string{
			"error": fmt.Sprintf("gateway error: %v", err),
		})
		return
	}

	writeNodeJSON(w, http.StatusOK, gwResp)
}

func writeNodeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
