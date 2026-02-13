package coordinator

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/snapek/claw-mesh/internal/types"
)

// Registry manages the set of known nodes.
type Registry struct {
	mu    sync.RWMutex
	nodes map[string]*types.Node
}

// NewRegistry creates an empty node registry.
func NewRegistry() *Registry {
	return &Registry{
		nodes: make(map[string]*types.Node),
	}
}

// Add registers a node. Returns error if the ID already exists.
func (r *Registry) Add(node *types.Node) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.nodes[node.ID]; exists {
		return fmt.Errorf("node %s already registered", node.ID)
	}
	r.nodes[node.ID] = node
	return nil
}

// Remove unregisters a node by ID.
func (r *Registry) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.nodes[id]; !exists {
		return false
	}
	delete(r.nodes, id)
	return true
}

// copyNode returns a deep copy of a Node.
func copyNode(n *types.Node) *types.Node {
	cp := *n
	if n.Capabilities.Tags != nil {
		cp.Capabilities.Tags = make([]string, len(n.Capabilities.Tags))
		copy(cp.Capabilities.Tags, n.Capabilities.Tags)
	}
	if n.Capabilities.Skills != nil {
		cp.Capabilities.Skills = make([]string, len(n.Capabilities.Skills))
		copy(cp.Capabilities.Skills, n.Capabilities.Skills)
	}
	return &cp
}

// Exists reports whether a node with the given ID is registered.
func (r *Registry) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.nodes[id]
	return ok
}

// Get returns a deep copy of a node by ID, or nil if not found.
func (r *Registry) Get(id string) *types.Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.nodes[id]
	if !ok {
		return nil
	}
	return copyNode(n)
}

// List returns deep copies of all registered nodes.
func (r *Registry) List() []*types.Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*types.Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		out = append(out, copyNode(n))
	}
	return out
}

// UpdateStatus sets a node's status. Returns false if node not found.
func (r *Registry) UpdateStatus(id string, status types.NodeStatus) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, exists := r.nodes[id]
	if !exists {
		return false
	}
	n.Status = status
	return true
}

// RecordHeartbeat updates a node's heartbeat time and status.
// Returns false if the node is not found.
func (r *Registry) RecordHeartbeat(nodeID string, status types.NodeStatus) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, exists := r.nodes[nodeID]
	if !exists {
		return false
	}
	n.LastHeartbeat = time.Now()
	n.Status = status
	return true
}

// MarkOfflineIfStale marks nodes offline if their last heartbeat exceeds timeout.
func (r *Registry) MarkOfflineIfStale(timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, n := range r.nodes {
		if n.Status == types.NodeStatusOffline {
			continue
		}
		if now.Sub(n.LastHeartbeat) > timeout {
			log.Printf("node %s (%s) missed heartbeat, marking offline", n.ID, n.Name)
			n.Status = types.NodeStatusOffline
		}
	}
}
