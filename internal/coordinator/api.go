package coordinator

import (
	"log"
	"net/http"
	"time"

	"github.com/snapek/claw-mesh/internal/types"
)

// handleRouteAuto handles POST /api/v1/route — auto-route a message.
func (s *Server) handleRouteAuto(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
		Source  string `json:"source"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}

	msgID, err := generateID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate message ID"})
		return
	}

	msg := &types.Message{
		ID:        msgID,
		Content:   req.Content,
		Source:    req.Source,
		CreatedAt: time.Now(),
	}

	node, err := s.router.Route(msg)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("routed message %s to node %s (%s)", msg.ID, node.ID, node.Name)
	writeJSON(w, http.StatusOK, types.MessageResponse{
		MessageID: msg.ID,
		NodeID:    node.ID,
		Response:  "routed",
	})
}

// handleRouteToNode handles POST /api/v1/route/{nodeId} — route to a specific node.
func (s *Server) handleRouteToNode(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("nodeId")

	var req struct {
		Content string `json:"content"`
		Source  string `json:"source"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}

	msgID, err := generateID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate message ID"})
		return
	}

	msg := &types.Message{
		ID:         msgID,
		Content:    req.Content,
		Source:     req.Source,
		TargetNode: nodeID,
		CreatedAt:  time.Now(),
	}

	node, err := s.router.Route(msg)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("routed message %s to node %s (%s)", msg.ID, node.ID, node.Name)
	writeJSON(w, http.StatusOK, types.MessageResponse{
		MessageID: msg.ID,
		NodeID:    node.ID,
		Response:  "routed",
	})
}

// handleListRules handles GET /api/v1/rules.
func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.router.ListRules())
}

// handleAddRule handles POST /api/v1/rules.
func (s *Server) handleAddRule(w http.ResponseWriter, r *http.Request) {
	var rule types.RoutingRule
	if err := decodeJSON(r, &rule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := s.router.AddRule(&rule); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to add rule"})
		return
	}

	log.Printf("routing rule added: %s", rule.ID)
	writeJSON(w, http.StatusCreated, rule)
}

// handleDeleteRule handles DELETE /api/v1/rules/{id}.
func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.router.RemoveRule(id) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "rule not found"})
		return
	}
	log.Printf("routing rule deleted: %s", id)
	w.WriteHeader(http.StatusNoContent)
}
