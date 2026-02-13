package coordinator

import (
	"fmt"
	"sync"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

// Router evaluates routing rules against the node registry to pick
// the best node for a given message.
type Router struct {
	mu       sync.RWMutex
	rules    []*types.RoutingRule
	registry *Registry
}

// NewRouter creates a router backed by the given registry.
func NewRouter(registry *Registry) *Router {
	return &Router{
		registry: registry,
	}
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
	rt.mu.Unlock()
	return nil
}

// RemoveRule deletes a rule by ID. Returns false if not found.
func (rt *Router) RemoveRule(id string) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for i, r := range rt.rules {
		if r.ID == id {
			rt.rules = append(rt.rules[:i], rt.rules[i+1:]...)
			return true
		}
	}
	return false
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
// it routes directly to that node. Otherwise it evaluates rules in order.
// Falls back to least-busy strategy if no rule matches.
func (rt *Router) Route(msg *types.Message) (*types.Node, error) {
	if msg.TargetNode != "" {
		node := rt.registry.Get(msg.TargetNode)
		if node == nil {
			return nil, fmt.Errorf("target node %q not found", msg.TargetNode)
		}
		if node.Status == types.NodeStatusOffline {
			return nil, fmt.Errorf("target node %q is offline", msg.TargetNode)
		}
		return node, nil
	}

	rt.mu.RLock()
	rules := make([]*types.RoutingRule, len(rt.rules))
	copy(rules, rt.rules)
	rt.mu.RUnlock()

	nodes := rt.registry.List()
	online := filterOnline(nodes)
	if len(online) == 0 {
		return nil, fmt.Errorf("no online nodes available")
	}

	// Evaluate rules in order.
	for _, rule := range rules {
		if isWildcard(rule) {
			return rt.applyStrategy(rule.Strategy, online)
		}
		candidates := matchNodes(rule, online)
		if len(candidates) == 0 {
			continue
		}
		// If rule targets a specific node name, prefer it.
		if rule.Target != "" {
			for _, n := range candidates {
				if n.Name == rule.Target || n.ID == rule.Target {
					return n, nil
				}
			}
			// Explicit target didn't match any candidate — skip this rule
			// instead of silently falling back to leastBusy.
			continue
		}
		return leastBusy(candidates), nil
	}

	// No rule matched — fall back to least-busy across all online nodes.
	return leastBusy(online), nil
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
