package coordinator

import (
	"fmt"
	"sync"

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

// Get returns a node by ID, or nil if not found.
func (r *Registry) Get(id string) *types.Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.nodes[id]
}

// List returns all registered nodes.
func (r *Registry) List() []*types.Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*types.Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		out = append(out, n)
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
