package coordinator

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SallyKAN/claw-mesh/internal/config"
	"github.com/SallyKAN/claw-mesh/internal/types"
)

// testAPI builds a minimal Server with an HTTP mux for API testing.
// It returns the test server and the admin token.
func testAPI(t *testing.T) (*httptest.Server, string) {
	t.Helper()

	token := "test-admin-token"
	reg := NewRegistry()
	rt := NewRouter(reg, nil, nil)
	fwd := NewForwarder()
	ts := NewTaskStore()

	s := &Server{
		cfg: &config.CoordinatorConfig{
			Token:        token,
			AllowPrivate: true,
		},
		registry:  reg,
		router:    rt,
		forwarder: fwd,
		taskStore: ts,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/nodes/register", s.requireAuth(s.handleRegister))
	mux.HandleFunc("DELETE /api/v1/nodes/{id}", s.requireAuth(s.handleDeregister))
	mux.HandleFunc("GET /api/v1/nodes", s.handleListNodes)
	mux.HandleFunc("GET /api/v1/nodes/{id}", s.handleGetNode)
	mux.HandleFunc("POST /api/v1/nodes/{id}/heartbeat", s.requireAuth(s.handleHeartbeat))
	mux.HandleFunc("POST /api/v1/route", s.requireAuth(s.handleRouteAuto))
	mux.HandleFunc("POST /api/v1/route/{nodeId}", s.requireAuth(s.handleRouteToNode))
	mux.HandleFunc("GET /api/v1/rules", s.handleListRules)
	mux.HandleFunc("POST /api/v1/rules", s.requireAuth(s.handleAddRule))
	mux.HandleFunc("DELETE /api/v1/rules/{id}", s.requireAuth(s.handleDeleteRule))

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, token
}

func doJSON(t *testing.T, method, url, token string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func decodeJSONBody(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestAPI_RegisterNode(t *testing.T) {
	srv, token := testAPI(t)

	body := types.RegisterRequest{
		Name:     "mac-mini",
		Endpoint: "192.168.1.100:9121",
		Capabilities: types.Capabilities{
			OS:   "darwin",
			Arch: "arm64",
			GPU:  true,
		},
	}

	resp := doJSON(t, "POST", srv.URL+"/api/v1/nodes/register", token, body)
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, b)
	}

	var regResp types.RegisterResponse
	decodeJSONBody(t, resp, &regResp)

	if regResp.NodeID == "" {
		t.Error("expected non-empty node ID")
	}
	if regResp.Token == "" {
		t.Error("expected non-empty node token")
	}

	// Verify node is listed.
	listResp := doJSON(t, "GET", srv.URL+"/api/v1/nodes", "", nil)
	defer listResp.Body.Close()
	var nodes []*types.Node
	decodeJSONBody(t, listResp, &nodes)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Name != "mac-mini" {
		t.Errorf("name = %q, want mac-mini", nodes[0].Name)
	}
}

func TestAPI_RegisterNode_MissingFields(t *testing.T) {
	srv, token := testAPI(t)

	// Missing name.
	resp := doJSON(t, "POST", srv.URL+"/api/v1/nodes/register", token, map[string]string{
		"endpoint": "192.168.1.100:9121",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", resp.StatusCode)
	}

	// Missing endpoint.
	resp = doJSON(t, "POST", srv.URL+"/api/v1/nodes/register", token, map[string]string{
		"name": "test-node",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for missing endpoint, got %d", resp.StatusCode)
	}
}

func TestAPI_AuthRequired(t *testing.T) {
	srv, _ := testAPI(t)

	// No token.
	resp := doJSON(t, "POST", srv.URL+"/api/v1/nodes/register", "", map[string]string{
		"name":     "test",
		"endpoint": "10.0.0.1:9121",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}

	// Wrong token.
	resp = doJSON(t, "POST", srv.URL+"/api/v1/nodes/register", "wrong-token", map[string]string{
		"name":     "test",
		"endpoint": "10.0.0.1:9121",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong token, got %d", resp.StatusCode)
	}
}

func TestAPI_Heartbeat(t *testing.T) {
	srv, token := testAPI(t)

	// Register a node first.
	regBody := types.RegisterRequest{
		Name:     "test-node",
		Endpoint: "192.168.1.100:9121",
	}
	regResp := doJSON(t, "POST", srv.URL+"/api/v1/nodes/register", token, regBody)
	var reg types.RegisterResponse
	decodeJSONBody(t, regResp, &reg)

	// Heartbeat with admin token.
	hbBody := types.HeartbeatRequest{Status: types.NodeStatusOnline}
	resp := doJSON(t, "POST", srv.URL+"/api/v1/nodes/"+reg.NodeID+"/heartbeat", token, hbBody)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}

	// Heartbeat with per-node token.
	resp = doJSON(t, "POST", srv.URL+"/api/v1/nodes/"+reg.NodeID+"/heartbeat", reg.Token, hbBody)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204 with node token, got %d", resp.StatusCode)
	}

	// Heartbeat for nonexistent node.
	resp = doJSON(t, "POST", srv.URL+"/api/v1/nodes/nonexistent/heartbeat", token, hbBody)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent node, got %d", resp.StatusCode)
	}
}

func TestAPI_Heartbeat_InvalidStatus(t *testing.T) {
	srv, token := testAPI(t)

	// Register a node first.
	regResp := doJSON(t, "POST", srv.URL+"/api/v1/nodes/register", token, types.RegisterRequest{
		Name:     "test-node",
		Endpoint: "192.168.1.100:9121",
	})
	var reg types.RegisterResponse
	decodeJSONBody(t, regResp, &reg)

	// Invalid status value.
	resp := doJSON(t, "POST", srv.URL+"/api/v1/nodes/"+reg.NodeID+"/heartbeat", token, map[string]string{
		"status": "invalid-status",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid status, got %d", resp.StatusCode)
	}
}

func TestAPI_GetNode(t *testing.T) {
	srv, token := testAPI(t)

	// Register.
	regResp := doJSON(t, "POST", srv.URL+"/api/v1/nodes/register", token, types.RegisterRequest{
		Name:     "mac-mini",
		Endpoint: "192.168.1.100:9121",
		Capabilities: types.Capabilities{
			OS:   "darwin",
			Arch: "arm64",
		},
	})
	var reg types.RegisterResponse
	decodeJSONBody(t, regResp, &reg)

	// Get by ID.
	resp := doJSON(t, "GET", srv.URL+"/api/v1/nodes/"+reg.NodeID, "", nil)
	var node types.Node
	decodeJSONBody(t, resp, &node)
	if node.Name != "mac-mini" {
		t.Errorf("name = %q, want mac-mini", node.Name)
	}

	// Get nonexistent.
	resp = doJSON(t, "GET", srv.URL+"/api/v1/nodes/nonexistent", "", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestAPI_DeregisterNode(t *testing.T) {
	srv, token := testAPI(t)

	// Register.
	regResp := doJSON(t, "POST", srv.URL+"/api/v1/nodes/register", token, types.RegisterRequest{
		Name:     "to-delete",
		Endpoint: "192.168.1.100:9121",
	})
	var reg types.RegisterResponse
	decodeJSONBody(t, regResp, &reg)

	// Delete.
	resp := doJSON(t, "DELETE", srv.URL+"/api/v1/nodes/"+reg.NodeID, token, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}

	// Verify gone.
	resp = doJSON(t, "GET", srv.URL+"/api/v1/nodes/"+reg.NodeID, "", nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after deletion, got %d", resp.StatusCode)
	}

	// Delete nonexistent.
	resp = doJSON(t, "DELETE", srv.URL+"/api/v1/nodes/nonexistent", token, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent, got %d", resp.StatusCode)
	}
}

func TestAPI_AddAndListRules(t *testing.T) {
	srv, token := testAPI(t)

	// Initially empty.
	resp := doJSON(t, "GET", srv.URL+"/api/v1/rules", "", nil)
	var rules []*types.RoutingRule
	decodeJSONBody(t, resp, &rules)
	if len(rules) != 0 {
		t.Fatalf("expected 0 rules initially, got %d", len(rules))
	}

	// Add a rule.
	rule := types.RoutingRule{
		Match:    types.MatchCriteria{RequiresOS: "linux"},
		Target:   "linux-dev",
		Strategy: "least-busy",
	}
	resp = doJSON(t, "POST", srv.URL+"/api/v1/rules", token, rule)
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, b)
	}
	var created types.RoutingRule
	decodeJSONBody(t, resp, &created)
	if created.ID == "" {
		t.Error("expected rule ID to be assigned")
	}

	// List should have 1 rule.
	resp = doJSON(t, "GET", srv.URL+"/api/v1/rules", "", nil)
	decodeJSONBody(t, resp, &rules)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Match.RequiresOS != "linux" {
		t.Errorf("rule requires_os = %q, want linux", rules[0].Match.RequiresOS)
	}
}

func TestAPI_AddRule_InvalidEmpty(t *testing.T) {
	srv, token := testAPI(t)

	// Empty match criteria should be rejected.
	resp := doJSON(t, "POST", srv.URL+"/api/v1/rules", token, types.RoutingRule{})
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty rule, got %d", resp.StatusCode)
	}
}

func TestAPI_DeleteRule(t *testing.T) {
	srv, token := testAPI(t)

	// Add a rule.
	rule := types.RoutingRule{
		Match:  types.MatchCriteria{RequiresOS: "linux"},
		Target: "linux-dev",
	}
	resp := doJSON(t, "POST", srv.URL+"/api/v1/rules", token, rule)
	var created types.RoutingRule
	decodeJSONBody(t, resp, &created)

	// Delete it.
	resp = doJSON(t, "DELETE", srv.URL+"/api/v1/rules/"+created.ID, token, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}

	// Should be empty now.
	resp = doJSON(t, "GET", srv.URL+"/api/v1/rules", "", nil)
	var rules []*types.RoutingRule
	decodeJSONBody(t, resp, &rules)
	if len(rules) != 0 {
		t.Errorf("expected 0 rules after deletion, got %d", len(rules))
	}

	// Delete nonexistent.
	resp = doJSON(t, "DELETE", srv.URL+"/api/v1/rules/nonexistent", token, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent rule, got %d", resp.StatusCode)
	}
}

func TestAPI_RouteAuto(t *testing.T) {
	srv, token := testAPI(t)

	// No nodes — should fail.
	resp := doJSON(t, "POST", srv.URL+"/api/v1/route", token, map[string]string{
		"content": "hello",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected 503 with no nodes, got %d", resp.StatusCode)
	}

	// Empty content — should fail.
	resp = doJSON(t, "POST", srv.URL+"/api/v1/route", token, map[string]string{
		"content": "",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for empty content, got %d", resp.StatusCode)
	}
}

func TestAPI_RouteToNode_NotFound(t *testing.T) {
	srv, token := testAPI(t)

	resp := doJSON(t, "POST", srv.URL+"/api/v1/route/nonexistent", token, map[string]string{
		"content": "hello",
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent node, got %d", resp.StatusCode)
	}
}
