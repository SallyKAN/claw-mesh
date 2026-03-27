package coordinator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

const decisionLogCap = 200

// Router evaluates routing rules against the node registry to pick
// the best node for a given message.
type Router struct {
	mu          sync.RWMutex
	rules       []*types.RoutingRule
	registry    *Registry
	store       *Store
	classifier  *LLMClassifier
	decisionsMu sync.RWMutex
	decisions   []*types.RoutingDecision
}

// NewRouter creates a router backed by the given registry.
// If store is non-nil, rules are loaded from and persisted to disk.
// If classifier is non-nil, LLM intent routing is enabled.
func NewRouter(registry *Registry, store *Store, classifier *LLMClassifier) *Router {
	rt := &Router{
		registry:   registry,
		classifier: classifier,
	}
	if store != nil {
		rt.store = store
		if rules, err := rt.store.LoadRules(); err != nil {
			log.Printf("WARN: failed to load persisted rules: %v", err)
		} else if len(rules) > 0 {
			rt.rules = rules
			log.Printf("loaded %d persisted routing rules", len(rules))
		}
	}
	return rt
}

// AddRule appends a routing rule and returns its assigned ID.
func (rt *Router) AddRule(rule *types.RoutingRule) error {
	id, err := generateID()
	if err != nil {
		return err
	}
	rule.ID = id
	rt.mu.Lock()
	rt.rules = append(rt.rules, rule)
	rules := make([]*types.RoutingRule, len(rt.rules))
	copy(rules, rt.rules)
	rt.mu.Unlock()
	if err := rt.persistRules(rules); err != nil {
		return fmt.Errorf("persisting rules: %w", err)
	}
	return nil
}

// RemoveRule deletes a rule by ID. Returns false if not found.
// Returns an error if persistence fails.
func (rt *Router) RemoveRule(id string) (bool, error) {
	rt.mu.Lock()
	found := false
	for i, r := range rt.rules {
		if r.ID == id {
			rt.rules = append(rt.rules[:i], rt.rules[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		rt.mu.Unlock()
		return false, nil
	}
	rules := make([]*types.RoutingRule, len(rt.rules))
	copy(rules, rt.rules)
	rt.mu.Unlock()
	if err := rt.persistRules(rules); err != nil {
		return true, fmt.Errorf("persisting rules: %w", err)
	}
	return true, nil
}

// persistRules saves rules to the store if configured.
func (rt *Router) persistRules(rules []*types.RoutingRule) error {
	if rt.store == nil {
		return nil
	}
	return rt.store.SaveRules(rules)
}

// ListRules returns a copy of all routing rules.
func (rt *Router) ListRules() []*types.RoutingRule {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	out := make([]*types.RoutingRule, len(rt.rules))
	for i, r := range rt.rules {
		cp := *r
		out[i] = &cp
	}
	return out
}

// Route picks the best node for a message. If msg.TargetNode is set,
// it routes directly to that node. Otherwise it tries LLM intent classification,
// then evaluates rules in order, and finally falls back to least-busy.
func (rt *Router) Route(msg *types.Message) (*types.Node, error) {
	if msg.TargetNode != "" {
		node := rt.registry.Get(msg.TargetNode)
		if node == nil {
			return nil, fmt.Errorf("target node %q not found", msg.TargetNode)
		}
		if node.Status == types.NodeStatusOffline {
			return nil, fmt.Errorf("target node %q is offline", msg.TargetNode)
		}
		rt.recordDecision(msg.Content, node.Name, "explicit", "")
		return node, nil
	}

	nodes := rt.registry.List()
	online := filterOnline(nodes)
	if len(online) == 0 {
		return nil, fmt.Errorf("no online nodes available")
	}

	// LLM intent classification — only when multiple nodes are available.
	if rt.classifier != nil && len(online) > 1 {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if result := rt.classifier.Classify(ctx, msg.Content, online); result != nil {
			for _, n := range online {
				if n.Name == result.NodeName {
					log.Printf("intent route: %q → %s (%s)", truncate(msg.Content, 40), n.Name, result.Reason)
					rt.recordDecision(msg.Content, n.Name, "llm", result.Reason)
					return n, nil
				}
			}
			// LLM returned a node name that's not online — fall through.
			log.Printf("intent route: LLM suggested %q but node not online, falling back", result.NodeName)
		}
	}

	rt.mu.RLock()
	rules := make([]*types.RoutingRule, len(rt.rules))
	copy(rules, rt.rules)
	rt.mu.RUnlock()

	// Evaluate rules in order.
	for _, rule := range rules {
		if isWildcard(rule) {
			node, err := rt.applyStrategy(rule.Strategy, online)
			if err == nil {
				rt.recordDecision(msg.Content, node.Name, "rule", rule.ID)
			}
			return node, err
		}
		candidates := matchNodes(rule, online)
		if len(candidates) == 0 {
			continue
		}
		if rule.Target != "" {
			for _, n := range candidates {
				if n.Name == rule.Target || n.ID == rule.Target {
					rt.recordDecision(msg.Content, n.Name, "rule", rule.ID)
					return n, nil
				}
			}
			continue
		}
		node := leastBusy(candidates)
		rt.recordDecision(msg.Content, node.Name, "rule", rule.ID)
		return node, nil
	}

	// No rule matched — fall back to least-busy across all online nodes.
	node := leastBusy(online)
	rt.recordDecision(msg.Content, node.Name, "least-busy", "")
	return node, nil
}

// recordDecision appends a routing decision to the ring buffer.
func (rt *Router) recordDecision(content, nodeName, method, reason string) {
	d := &types.RoutingDecision{
		Timestamp:   time.Now(),
		MessageSnip: truncate(content, 60),
		NodeName:    nodeName,
		Method:      method,
		Reason:      reason,
	}
	rt.decisionsMu.Lock()
	rt.decisions = append(rt.decisions, d)
	if len(rt.decisions) > decisionLogCap {
		rt.decisions = rt.decisions[len(rt.decisions)-decisionLogCap:]
	}
	rt.decisionsMu.Unlock()
}

// ListDecisions returns a copy of the recent routing decisions (newest last).
func (rt *Router) ListDecisions() []*types.RoutingDecision {
	rt.decisionsMu.RLock()
	defer rt.decisionsMu.RUnlock()
	out := make([]*types.RoutingDecision, len(rt.decisions))
	copy(out, rt.decisions)
	return out
}

// truncate returns the first n runes of s, appending "…" if truncated.
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n]) + "…"
}

// filterOnline returns nodes that are not offline.
func filterOnline(nodes []*types.Node) []*types.Node {
	var out []*types.Node
	for _, n := range nodes {
		if n.Status != types.NodeStatusOffline {
			out = append(out, n)
		}
	}
	return out
}

// isWildcard returns true if the rule's match criteria is a wildcard.
func isWildcard(rule *types.RoutingRule) bool {
	if rule.Match.Wildcard != nil && *rule.Match.Wildcard {
		return true
	}
	return false
}

// matchNodes filters nodes that satisfy a rule's match criteria.
func matchNodes(rule *types.RoutingRule, nodes []*types.Node) []*types.Node {
	var out []*types.Node
	for _, n := range nodes {
		if matchesCriteria(&rule.Match, n) {
			out = append(out, n)
		}
	}
	return out
}

// matchesCriteria checks whether a single node satisfies the criteria.
func matchesCriteria(mc *types.MatchCriteria, n *types.Node) bool {
	if mc.RequiresGPU != nil && *mc.RequiresGPU && !n.Capabilities.GPU {
		return false
	}
	if mc.RequiresOS != "" && mc.RequiresOS != n.Capabilities.OS {
		return false
	}
	if mc.RequiresSkill != "" && !hasSkill(n, mc.RequiresSkill) {
		return false
	}
	return true
}

// hasSkill checks if a node advertises a given skill or tag.
func hasSkill(n *types.Node, skill string) bool {
	for _, s := range n.Capabilities.Skills {
		if s == skill {
			return true
		}
	}
	for _, t := range n.Capabilities.Tags {
		if t == skill {
			return true
		}
	}
	return false
}

// applyStrategy selects a node using the named strategy.
func (rt *Router) applyStrategy(strategy string, nodes []*types.Node) (*types.Node, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes available for strategy %q", strategy)
	}
	// Default and "least-busy" both use the same logic.
	return leastBusy(nodes), nil
}

// leastBusy picks the node that is least loaded.
// Prefers "online" over "busy".
func leastBusy(nodes []*types.Node) *types.Node {
	var best *types.Node
	for _, n := range nodes {
		if best == nil {
			best = n
			continue
		}
		if statusPriority(n.Status) < statusPriority(best.Status) {
			best = n
		}
	}
	return best
}

// statusPriority returns a numeric priority (lower = more available).
func statusPriority(s types.NodeStatus) int {
	switch s {
	case types.NodeStatusOnline:
		return 0
	case types.NodeStatusBusy:
		return 1
	default:
		return 2
	}
}
