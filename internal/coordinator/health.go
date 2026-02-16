package coordinator

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// HealthChecker monitors node heartbeats and optionally probes node endpoints.
type HealthChecker struct {
	registry *Registry
	timeout  time.Duration
	interval time.Duration
	// Active probing
	activeProbe    bool
	probeClient    *http.Client
	failThreshold  int
	probeFailures  map[string]int // nodeID -> consecutive probe failures
	probeFailuresMu sync.Mutex

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
		registry:      registry,
		timeout:       timeout,
		interval:      interval,
		activeProbe:   true,
		probeClient:   &http.Client{Timeout: 5 * time.Second},
		failThreshold: 2,
		probeFailures: make(map[string]int),
		stopCh:        make(chan struct{}),
		done:          make(chan struct{}),
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
			if h.activeProbe {
				h.probeNodes()
			}
		}
	}
}

// probeNodes sends HTTP GET /healthz to each online node endpoint.
func (h *HealthChecker) probeNodes() {
	nodes := h.registry.List()
	for _, n := range nodes {
		if n.Status == "offline" {
			continue
		}
		url := fmt.Sprintf("http://%s/healthz", n.Endpoint)
		resp, err := h.probeClient.Get(url)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			h.recordProbeFailure(n.ID, n.Name)
		} else {
			resp.Body.Close()
			h.clearProbeFailure(n.ID)
		}
	}
}

func (h *HealthChecker) recordProbeFailure(nodeID, name string) {
	h.probeFailuresMu.Lock()
	defer h.probeFailuresMu.Unlock()
	h.probeFailures[nodeID]++
	count := h.probeFailures[nodeID]
	if count >= h.failThreshold {
		log.Printf("node %s (%s) failed %d active probes, marking offline", nodeID, name, count)
		h.registry.UpdateStatus(nodeID, "offline")
		delete(h.probeFailures, nodeID)
	}
}

func (h *HealthChecker) clearProbeFailure(nodeID string) {
	h.probeFailuresMu.Lock()
	defer h.probeFailuresMu.Unlock()
	delete(h.probeFailures, nodeID)
}
