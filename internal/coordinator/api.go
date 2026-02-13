package coordinator

import (
	"fmt"
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
	if err := decodeJSON(w, r, &req); err != nil {
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

	log.Printf("forwarding message %s to node %s (%s)", msg.ID, node.ID, node.Name)
	nodeToken := s.registry.GetNodeToken(node.ID)
	fwdResp, err := s.forwarder.ForwardMessage(r.Context(), node, msg, nodeToken)
	if err != nil {
		log.Printf("forward failed for message %s: %v", msg.ID, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("forwarding failed: %v", err)})
		return
	}
	fwdResp.NodeID = node.ID
	writeJSON(w, http.StatusOK, fwdResp)
}

// handleRouteToNode handles POST /api/v1/route/{nodeId} — route to a specific node.
func (s *Server) handleRouteToNode(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("nodeId")

	var req struct {
		Content string `json:"content"`
		Source  string `json:"source"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
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
		// Use 502 for offline nodes, 503 for unavailable, 404 for not found.
		n := s.registry.Get(nodeID)
		if n == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		} else if n.Status == types.NodeStatusOffline {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		} else {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		}
		return
	}

	log.Printf("forwarding message %s to node %s (%s)", msg.ID, node.ID, node.Name)
	nodeToken := s.registry.GetNodeToken(node.ID)
	fwdResp, err := s.forwarder.ForwardMessage(r.Context(), node, msg, nodeToken)
	if err != nil {
		log.Printf("forward failed for message %s: %v", msg.ID, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("forwarding failed: %v", err)})
		return
	}
	fwdResp.NodeID = node.ID
	writeJSON(w, http.StatusOK, fwdResp)
}

// handleListRules handles GET /api/v1/rules.
func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.router.ListRules())
}

// handleAddRule handles POST /api/v1/rules.
func (s *Server) handleAddRule(w http.ResponseWriter, r *http.Request) {
	var rule types.RoutingRule
	if err := decodeJSON(w, r, &rule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := validateRule(&rule); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
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

// validStrategies lists the accepted routing strategy values.
var validStrategies = map[string]bool{
	"":           true,
	"least-busy": true,
}

// validateRule checks a routing rule for invalid or contradictory fields.
func validateRule(rule *types.RoutingRule) error {
	isWild := rule.Match.Wildcard != nil && *rule.Match.Wildcard
	hasCriteria := rule.Match.RequiresGPU != nil || rule.Match.RequiresOS != "" || rule.Match.RequiresSkill != ""

	// Reject empty criteria (no match fields at all).
	if !isWild && !hasCriteria {
		return fmt.Errorf("rule must have at least one match criterion or be a wildcard")
	}

	// Reject wildcard combined with specific criteria.
	if isWild && hasCriteria {
		return fmt.Errorf("wildcard rule cannot have other match criteria")
	}

	// Reject wildcard combined with a specific target.
	if isWild && rule.Target != "" {
		return fmt.Errorf("wildcard rule cannot specify a target node")
	}

	// Validate strategy value.
	if !validStrategies[rule.Strategy] {
		return fmt.Errorf("invalid strategy %q; valid values: least-busy", rule.Strategy)
	}

	return nil
}
