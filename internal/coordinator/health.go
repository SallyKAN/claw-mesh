package coordinator

import (
	"sync"
	"time"
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
			h.registry.MarkOfflineIfStale(h.timeout)
		}
	}
}
