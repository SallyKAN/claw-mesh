package node

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/snapek/claw-mesh/internal/types"
)

// Handler serves the node-side HTTP API for receiving forwarded messages.
type Handler struct {
	mux *http.ServeMux
}

// NewHandler creates a node message handler.
func NewHandler() *Handler {
	h := &Handler{
		mux: http.NewServeMux(),
	}
	h.mux.HandleFunc("POST /api/v1/messages", h.handleMessage)
	return h
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// handleMessage receives a forwarded message from the coordinator.
// For now it echoes back the content as a placeholder for OpenClaw gateway integration.
func (h *Handler) handleMessage(w http.ResponseWriter, r *http.Request) {
	var msg types.Message
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&msg); err != nil {
		writeNodeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid message body"})
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
