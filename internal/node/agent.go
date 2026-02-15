package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

const heartbeatInterval = 15 * time.Second

// Agent is the node-side sidecar that registers with the coordinator,
// sends heartbeats, and handles graceful shutdown.
type Agent struct {
	coordinatorURL string
	token          string
	name           string
	endpoint       string
	capabilities   types.Capabilities

	gatewayEndpoint string
	gatewayToken    string
	gatewayTimeout  int

	nodeID string
	client *http.Client

	listenAddr string
	httpServer *http.Server

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	done      chan struct{}
	started   bool
}

// AgentConfig holds the parameters needed to create an Agent.
type AgentConfig struct {
	CoordinatorURL  string
	Token           string
	Name            string
	Endpoint        string
	Tags            []string
	ListenAddr      string // address for the local message handler (default: :9121)
	GatewayEndpoint string // OpenClaw Gateway endpoint (default: auto-discover)
	GatewayToken    string // OpenClaw Gateway auth token
	GatewayTimeout  int    // Gateway request timeout in seconds (default: 120)
}

// NewAgent creates a node agent with the given configuration.
func NewAgent(cfg AgentConfig) *Agent {
	caps := DetectCapabilities(cfg.Tags)
	listenAddr := cfg.ListenAddr
	if listenAddr == "" {
		listenAddr = ":9121"
	}
	return &Agent{
		coordinatorURL:  cfg.CoordinatorURL,
		token:           cfg.Token,
		name:            cfg.Name,
		endpoint:        cfg.Endpoint,
		capabilities:    caps,
		gatewayEndpoint: cfg.GatewayEndpoint,
		gatewayToken:    cfg.GatewayToken,
		gatewayTimeout:  cfg.GatewayTimeout,
		client:          &http.Client{Timeout: 10 * time.Second},
		listenAddr:      listenAddr,
		stopCh:          make(chan struct{}),
		done:            make(chan struct{}),
	}
}

// Register sends a registration request to the coordinator.
func (a *Agent) Register() error {
	req := types.RegisterRequest{
		Name:         a.name,
		Endpoint:     a.endpoint,
		Capabilities: a.capabilities,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshaling register request: %w", err)
	}

	url := a.coordinatorURL + "/api/v1/nodes/register"
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating register request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if a.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.token)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("registering with coordinator: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("registration failed (%d): %s", resp.StatusCode, errResp.Error)
	}

	var regResp types.RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("decoding register response: %w", err)
	}

	a.nodeID = regResp.NodeID
	if regResp.Token != "" {
		a.token = regResp.Token
	}

	log.Printf("registered as node %s", a.nodeID)
	return nil
}

// StartHeartbeat begins sending periodic heartbeats to the coordinator.
// Safe to call multiple times; only the first call starts the loop.
func (a *Agent) StartHeartbeat() {
	a.startOnce.Do(func() {
		a.started = true
		go a.heartbeatLoop()
	})
}

// StartHandler starts the local HTTP server for receiving forwarded messages.
func (a *Agent) StartHandler() error {
	var gw GatewayClient
	if a.gatewayEndpoint != "" {
		gwToken := ResolveGatewayToken(a.gatewayToken, "")
		gw = NewHTTPGatewayClient(a.gatewayEndpoint, gwToken, a.gatewayTimeout)
		log.Printf("gateway client configured: %s", a.gatewayEndpoint)
	} else {
		log.Printf("WARN: no gateway endpoint configured, messages will be echoed")
	}
	handler := NewHandler(&a.token, gw)
	a.httpServer = &http.Server{
		Addr:    a.listenAddr,
		Handler: handler,
	}
	ln, err := net.Listen("tcp", a.listenAddr)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", a.listenAddr, err)
	}
	log.Printf("node handler listening on %s", a.listenAddr)
	go a.httpServer.Serve(ln)
	return nil
}

func (a *Agent) heartbeatLoop() {
	defer close(a.done)
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			if err := a.sendHeartbeat(); err != nil {
				log.Printf("heartbeat failed: %v", err)
			}
		}
	}
}

func (a *Agent) sendHeartbeat() error {
	req := types.HeartbeatRequest{
		Status: types.NodeStatusOnline,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/api/v1/nodes/%s/heartbeat", a.coordinatorURL, a.nodeID)
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if a.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.token)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("heartbeat returned status %d", resp.StatusCode)
	}
	return nil
}

// Shutdown deregisters the node and stops the heartbeat loop.
// Safe to call even if StartHeartbeat was never called.
func (a *Agent) Shutdown() {
	a.stopOnce.Do(func() {
		close(a.stopCh)
	})

	// Only wait for heartbeat loop if it was started.
	if a.started {
		<-a.done
	}

	// Stop the local HTTP server if running.
	if a.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.httpServer.Shutdown(ctx)
	}

	// Deregister from coordinator.
	if a.nodeID != "" {
		a.deregister()
	}
}

func (a *Agent) deregister() {
	url := fmt.Sprintf("%s/api/v1/nodes/%s", a.coordinatorURL, a.nodeID)
	httpReq, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		log.Printf("failed to create deregister request: %v", err)
		return
	}
	if a.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+a.token)
	}

	resp, err := a.client.Do(httpReq)
	if err != nil {
		log.Printf("failed to deregister: %v", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("deregister returned unexpected status %d for node %s", resp.StatusCode, a.nodeID)
		return
	}
	log.Printf("deregistered node %s", a.nodeID)
}
