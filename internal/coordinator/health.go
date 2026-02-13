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

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	done      chan struct{}
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
		done:     make(chan struct{}),
	}
}

// Start begins the background health check loop.
// Safe to call multiple times; only the first call starts the loop.
func (h *HealthChecker) Start() {
	h.startOnce.Do(func() {
		go h.loop()
	})
}

// Stop terminates the health check loop and waits for it to finish.
func (h *HealthChecker) Stop() {
	h.stopOnce.Do(func() {
		close(h.stopCh)
	})
	<-h.done
}

func (h *HealthChecker) loop() {
	defer close(h.done)
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
