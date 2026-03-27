package coordinator

import (
	"testing"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

// helper to create a bool pointer.
func boolPtr(b bool) *bool { return &b }

// makeNode creates a test node with sensible defaults.
func makeNode(id, name, os, arch string, gpu bool, status types.NodeStatus, skills, tags []string) *types.Node {
	return &types.Node{
		ID:       id,
		Name:     name,
		Endpoint: "127.0.0.1:9121",
		Capabilities: types.Capabilities{
			OS:     os,
			Arch:   arch,
			GPU:    gpu,
			Tags:   tags,
			Skills: skills,
		},
		Status:        status,
		LastHeartbeat: time.Now(),
	}
}

func TestRoute_TargetNode_Direct(t *testing.T) {
	reg := NewRegistry()
	node := makeNode("n1", "mac-mini", "darwin", "arm64", true, types.NodeStatusOnline, nil, nil)
	reg.Add(node)

	rt := NewRouter(reg, nil, nil)
	msg := &types.Message{TargetNode: "n1", Content: "hello"}

	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "n1" {
		t.Errorf("expected node n1, got %s", got.ID)
	}
}

func TestRoute_TargetNode_NotFound(t *testing.T) {
	reg := NewRegistry()
	rt := NewRouter(reg, nil, nil)
	msg := &types.Message{TargetNode: "nonexistent", Content: "hello"}

	_, err := rt.Route(msg)
	if err == nil {
		t.Fatal("expected error for nonexistent target node")
	}
}

func TestRoute_TargetNode_Offline(t *testing.T) {
	reg := NewRegistry()
	node := makeNode("n1", "mac-mini", "darwin", "arm64", true, types.NodeStatusOffline, nil, nil)
	reg.Add(node)

	rt := NewRouter(reg, nil, nil)
	msg := &types.Message{TargetNode: "n1", Content: "hello"}

	_, err := rt.Route(msg)
	if err == nil {
		t.Fatal("expected error for offline target node")
	}
}

func TestRoute_GPURule_MatchesTarget(t *testing.T) {
	reg := NewRegistry()
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", true, types.NodeStatusOnline, nil, nil))
	reg.Add(makeNode("n2", "linux-dev", "linux", "amd64", false, types.NodeStatusOnline, nil, nil))

	rt := NewRouter(reg, nil, nil)
	rt.rules = []*types.RoutingRule{
		{ID: "r1", Match: types.MatchCriteria{RequiresGPU: boolPtr(true)}, Target: "mac-mini"},
	}

	msg := &types.Message{Content: "generate image"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "n1" {
		t.Errorf("expected GPU node n1, got %s", got.ID)
	}
}

func TestRoute_OSRule_MatchesDarwin(t *testing.T) {
	reg := NewRegistry()
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", true, types.NodeStatusOnline, nil, nil))
	reg.Add(makeNode("n2", "linux-dev", "linux", "amd64", false, types.NodeStatusOnline, nil, nil))

	rt := NewRouter(reg, nil, nil)
	rt.rules = []*types.RoutingRule{
		{ID: "r1", Match: types.MatchCriteria{RequiresOS: "darwin"}, Target: "mac-mini"},
	}

	msg := &types.Message{Content: "run xcode build"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "n1" {
		t.Errorf("expected darwin node n1, got %s", got.ID)
	}
}

func TestRoute_SkillRule_MatchesDocker(t *testing.T) {
	reg := NewRegistry()
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", true, types.NodeStatusOnline, nil, nil))
	reg.Add(makeNode("n2", "linux-dev", "linux", "amd64", false, types.NodeStatusOnline, []string{"docker", "python"}, nil))

	rt := NewRouter(reg, nil, nil)
	rt.rules = []*types.RoutingRule{
		{ID: "r1", Match: types.MatchCriteria{RequiresSkill: "docker"}, Target: "linux-dev"},
	}

	msg := &types.Message{Content: "docker build"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "n2" {
		t.Errorf("expected docker node n2, got %s", got.ID)
	}
}

func TestRoute_SkillRule_MatchesTag(t *testing.T) {
	reg := NewRegistry()
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", true, types.NodeStatusOnline, nil, nil))
	reg.Add(makeNode("n2", "linux-dev", "linux", "amd64", false, types.NodeStatusOnline, nil, []string{"ubuntu", "laptop"}))

	rt := NewRouter(reg, nil, nil)
	rt.rules = []*types.RoutingRule{
		{ID: "r1", Match: types.MatchCriteria{RequiresSkill: "ubuntu"}, Target: "linux-dev"},
	}

	msg := &types.Message{Content: "apt install"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "n2" {
		t.Errorf("expected tagged node n2, got %s", got.ID)
	}
}

func TestRoute_RuleOrder_FirstWins(t *testing.T) {
	reg := NewRegistry()
	// Both nodes have GPU
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", true, types.NodeStatusOnline, nil, nil))
	reg.Add(makeNode("n2", "linux-gpu", "linux", "amd64", true, types.NodeStatusOnline, nil, nil))

	rt := NewRouter(reg, nil, nil)
	rt.rules = []*types.RoutingRule{
		{ID: "r1", Match: types.MatchCriteria{RequiresGPU: boolPtr(true)}, Target: "mac-mini"},
		{ID: "r2", Match: types.MatchCriteria{RequiresGPU: boolPtr(true)}, Target: "linux-gpu"},
	}

	msg := &types.Message{Content: "gpu task"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "n1" {
		t.Errorf("expected first matching rule target n1, got %s", got.ID)
	}
}

func TestRoute_NoRuleMatch_FallbackLeastBusy(t *testing.T) {
	reg := NewRegistry()
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", false, types.NodeStatusBusy, nil, nil))
	reg.Add(makeNode("n2", "linux-dev", "linux", "amd64", false, types.NodeStatusOnline, nil, nil))

	rt := NewRouter(reg, nil, nil)
	// Rule that matches nothing (requires GPU but neither has it)
	rt.rules = []*types.RoutingRule{
		{ID: "r1", Match: types.MatchCriteria{RequiresGPU: boolPtr(true)}, Target: "nonexistent"},
	}

	msg := &types.Message{Content: "generic task"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fall back to least-busy (n2 is online, n1 is busy)
	if got.ID != "n2" {
		t.Errorf("expected least-busy fallback to n2, got %s", got.ID)
	}
}

func TestRoute_WildcardRule(t *testing.T) {
	reg := NewRegistry()
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", false, types.NodeStatusBusy, nil, nil))
	reg.Add(makeNode("n2", "linux-dev", "linux", "amd64", false, types.NodeStatusOnline, nil, nil))

	rt := NewRouter(reg, nil, nil)
	rt.rules = []*types.RoutingRule{
		{ID: "r1", Match: types.MatchCriteria{Wildcard: boolPtr(true)}, Strategy: "least-busy"},
	}

	msg := &types.Message{Content: "anything"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Wildcard with least-busy should pick the online node
	if got.ID != "n2" {
		t.Errorf("expected wildcard least-busy to pick n2, got %s", got.ID)
	}
}

// TestRoute_LLMClassifier_NoLocalNode verifies that when no local node is
// registered (i.e. no 127.0.0.1 endpoint), LLM classification is skipped
// and routing falls back to least-busy without error.
func TestRoute_LLMClassifier_NoLocalNode(t *testing.T) {
	reg := NewRegistry()
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", false, types.NodeStatusOnline, nil, nil))
	reg.Add(makeNode("n2", "linux-dev", "linux", "amd64", false, types.NodeStatusOnline, nil, nil))

	rt := NewRouter(reg, nil, nil)

	msg := &types.Message{Content: "please run this on linux-dev"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No local node → no LLM → least-busy returns first online node (n1).
	if got == nil {
		t.Fatal("expected a node, got nil")
	}
}

// TestRoute_LLMClassifier_FallbackOnUnavailable verifies that when the LLM
// is unavailable the router still returns a node via least-busy.
func TestRoute_LLMClassifier_FallbackOnUnavailable(t *testing.T) {
	reg := NewRegistry()
	// Register a "local" node so localClassifier() finds it.
	localNode := makeNode("local", "local-node", "darwin", "arm64", false, types.NodeStatusOnline, nil, nil)
	localNode.Endpoint = "127.0.0.1:9999" // unreachable port — LLM call will fail
	reg.Add(localNode)
	reg.Add(makeNode("n2", "linux-dev", "linux", "amd64", false, types.NodeStatusOnline, nil, nil))

	rt := NewRouter(reg, nil, nil)

	msg := &types.Message{Content: "帮我启动 docker 容器"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// LLM call fails (port 9999 unreachable) → falls back to least-busy.
	if got == nil {
		t.Fatal("expected a node from fallback, got nil")
	}
}

func TestRoute_SmartRouting_Ambiguous(t *testing.T) {
	reg := NewRegistry()
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", false, types.NodeStatusOnline, nil, nil))
	reg.Add(makeNode("n2", "linux-dev", "linux", "amd64", false, types.NodeStatusOnline, nil, nil))

	rt := NewRouter(reg, nil, nil)

	// Message mentions both node names — should NOT do smart routing
	msg := &types.Message{Content: "copy file from mac-mini to linux-dev"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fall back to least-busy (either node is fine, just not smart-routed)
	if got == nil {
		t.Fatal("expected a node from fallback, got nil")
	}
}

func TestRoute_NoOnlineNodes(t *testing.T) {
	reg := NewRegistry()
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", false, types.NodeStatusOffline, nil, nil))
	reg.Add(makeNode("n2", "linux-dev", "linux", "amd64", false, types.NodeStatusOffline, nil, nil))

	rt := NewRouter(reg, nil, nil)
	msg := &types.Message{Content: "hello"}

	_, err := rt.Route(msg)
	if err == nil {
		t.Fatal("expected error when no nodes are online")
	}
}

func TestRoute_EmptyRegistry(t *testing.T) {
	reg := NewRegistry()
	rt := NewRouter(reg, nil, nil)
	msg := &types.Message{Content: "hello"}

	_, err := rt.Route(msg)
	if err == nil {
		t.Fatal("expected error when registry is empty")
	}
}

func TestLeastBusy_PrefersOnlineOverBusy(t *testing.T) {
	nodes := []*types.Node{
		makeNode("n1", "busy-node", "linux", "amd64", false, types.NodeStatusBusy, nil, nil),
		makeNode("n2", "online-node", "linux", "amd64", false, types.NodeStatusOnline, nil, nil),
	}

	got := leastBusy(nodes)
	if got.ID != "n2" {
		t.Errorf("expected online node n2, got %s (status=%s)", got.ID, got.Status)
	}
}

func TestLeastBusy_AllBusy(t *testing.T) {
	nodes := []*types.Node{
		makeNode("n1", "busy-1", "linux", "amd64", false, types.NodeStatusBusy, nil, nil),
		makeNode("n2", "busy-2", "linux", "amd64", false, types.NodeStatusBusy, nil, nil),
	}

	got := leastBusy(nodes)
	if got == nil {
		t.Fatal("expected a node even when all are busy")
	}
}

func TestMatchesCriteria_Combined(t *testing.T) {
	node := makeNode("n1", "gpu-linux", "linux", "amd64", true, types.NodeStatusOnline, []string{"docker"}, nil)

	// All criteria satisfied
	mc := &types.MatchCriteria{
		RequiresGPU:   boolPtr(true),
		RequiresOS:    "linux",
		RequiresSkill: "docker",
	}
	if !matchesCriteria(mc, node) {
		t.Error("expected node to match all combined criteria")
	}

	// One criterion fails (wrong OS)
	mc2 := &types.MatchCriteria{
		RequiresGPU: boolPtr(true),
		RequiresOS:  "darwin",
	}
	if matchesCriteria(mc2, node) {
		t.Error("expected node NOT to match when OS doesn't match")
	}

	// GPU required but node has no GPU
	nodeNoGPU := makeNode("n2", "linux-cpu", "linux", "amd64", false, types.NodeStatusOnline, nil, nil)
	mc3 := &types.MatchCriteria{RequiresGPU: boolPtr(true)}
	if matchesCriteria(mc3, nodeNoGPU) {
		t.Error("expected node NOT to match when GPU is required but absent")
	}
}

func TestRoute_RuleTargetByID(t *testing.T) {
	reg := NewRegistry()
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", true, types.NodeStatusOnline, nil, nil))

	rt := NewRouter(reg, nil, nil)
	rt.rules = []*types.RoutingRule{
		{ID: "r1", Match: types.MatchCriteria{RequiresOS: "darwin"}, Target: "n1"}, // target by ID
	}

	msg := &types.Message{Content: "test"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "n1" {
		t.Errorf("expected routing by node ID n1, got %s", got.ID)
	}
}

func TestRoute_RuleTargetNotOnline_SkipsRule(t *testing.T) {
	reg := NewRegistry()
	reg.Add(makeNode("n1", "mac-mini", "darwin", "arm64", true, types.NodeStatusOffline, nil, nil))
	reg.Add(makeNode("n2", "linux-dev", "linux", "amd64", false, types.NodeStatusOnline, nil, nil))

	rt := NewRouter(reg, nil, nil)
	rt.rules = []*types.RoutingRule{
		{ID: "r1", Match: types.MatchCriteria{RequiresOS: "darwin"}, Target: "mac-mini"},
	}

	msg := &types.Message{Content: "test"}
	got, err := rt.Route(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// darwin rule can't match any online node, falls through to least-busy
	if got.ID != "n2" {
		t.Errorf("expected fallback to n2, got %s", got.ID)
	}
}

func TestAddRule_AssignsID(t *testing.T) {
	reg := NewRegistry()
	rt := NewRouter(reg, nil, nil)

	rule := &types.RoutingRule{
		Match:  types.MatchCriteria{RequiresOS: "linux"},
		Target: "linux-dev",
	}
	if err := rt.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}
	if rule.ID == "" {
		t.Error("expected rule ID to be assigned")
	}

	rules := rt.ListRules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != rule.ID {
		t.Errorf("listed rule ID %s != added rule ID %s", rules[0].ID, rule.ID)
	}
}

func TestRemoveRule(t *testing.T) {
	reg := NewRegistry()
	rt := NewRouter(reg, nil, nil)

	rule := &types.RoutingRule{Match: types.MatchCriteria{RequiresOS: "linux"}, Target: "x"}
	rt.AddRule(rule)

	found, err := rt.RemoveRule(rule.ID)
	if err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}
	if !found {
		t.Error("expected rule to be found and removed")
	}
	if len(rt.ListRules()) != 0 {
		t.Error("expected no rules after removal")
	}

	found, err = rt.RemoveRule("nonexistent")
	if err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}
	if found {
		t.Error("expected nonexistent rule not to be found")
	}
}
