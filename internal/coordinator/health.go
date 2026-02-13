package coordinator

import (
	"log"
	"sync"
	"time"

	"github.com/snapek/claw-mesh/internal/types"
)

// HealthChecker monitors node heartbeats and marks stale nodes offline.
type HealthChecker struct {
	registry *Registry
	timeout  time.Duration
	interval time.Duration

	stopOnce sync.Once
	stopCh   chan struct{}
}

// NewHealthChecker creates a health checker.
// timeout is how long since last heartbeat before a node is marked offline.
// interval is how often the checker runs.
func NewHealthChecker(registry *Registry, timeout, interval time.Duration) *HealthChecker {
	return &HealthChecker{
		registry: registry,
		timeout:  timeout,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// RecordHeartbeat updates a node's last heartbeat time and status.
func (h *HealthChecker) RecordHeartbeat(nodeID string, status types.NodeStatus) bool {
	h.registry.mu.Lock()
	defer h.registry.mu.Unlock()
	n, exists := h.registry.nodes[nodeID]
	if !exists {
		return false
	}
	n.LastHeartbeat = time.Now()
	n.Status = status
	return true
}

// Start begins the background health check loop.
func (h *HealthChecker) Start() {
	go h.loop()
}

// Stop terminates the health check loop.
func (h *HealthChecker) Stop() {
	h.stopOnce.Do(func() {
		close(h.stopCh)
	})
}

func (h *HealthChecker) loop() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		select {
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.check()
		}
	}
}

func (h *HealthChecker) check() {
	now := time.Now()
	for _, node := range h.registry.List() {
		if node.Status == types.NodeStatusOffline {
			continue
		}
		if now.Sub(node.LastHeartbeat) > h.timeout {
			log.Printf("node %s (%s) missed heartbeat, marking offline", node.ID, node.Name)
			h.registry.UpdateStatus(node.ID, types.NodeStatusOffline)
		}
	}
}
