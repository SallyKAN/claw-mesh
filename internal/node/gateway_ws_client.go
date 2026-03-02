package node

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
	"github.com/gorilla/websocket"
)

type wsError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type wsPending struct {
	ch chan wsResponse
}

type wsResponse struct {
	ok      bool
	payload json.RawMessage
	err     string
}

// agentRun tracks an in-flight agent run, collecting streamed text.
type agentRun struct {
	text string
	done chan struct{}
	err  string
}

// WSGatewayClient talks to an OpenClaw Gateway via WebSocket RPC.
type WSGatewayClient struct {
	endpoint string
	token    string
	timeout  time.Duration

	mu      sync.Mutex
	conn    *websocket.Conn
	pending map[string]*wsPending
	runs    map[string]*agentRun // runId -> agentRun
	ready   chan struct{}
	seq     int
}

// NewWSGatewayClient creates a WebSocket-based gateway client.
func NewWSGatewayClient(endpoint, token string, timeoutSec int) *WSGatewayClient {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	return &WSGatewayClient{
		endpoint: endpoint,
		token:    token,
		timeout:  time.Duration(timeoutSec) * time.Second,
		pending:  make(map[string]*wsPending),
		runs:     make(map[string]*agentRun),
	}
}

// Connect establishes the WebSocket connection and completes the handshake.
func (c *WSGatewayClient) Connect(ctx context.Context) error {
	url := "ws://" + c.endpoint
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return fmt.Errorf("ws dial %s: %w", url, err)
	}
	c.conn = conn
	c.ready = make(chan struct{})

	go c.readLoop()

	select {
	case <-c.ready:
		return nil
	case <-ctx.Done():
		conn.Close()
		return ctx.Err()
	case <-time.After(15 * time.Second):
		conn.Close()
		return fmt.Errorf("gateway handshake timeout")
	}
}

func (c *WSGatewayClient) readLoop() {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			for _, p := range c.pending {
				p.ch <- wsResponse{err: "connection closed"}
			}
			c.pending = make(map[string]*wsPending)
			for _, r := range c.runs {
				r.err = "connection closed"
				select {
				case <-r.done:
				default:
					close(r.done)
				}
			}
			c.mu.Unlock()
			return
		}
		c.handleMessage(msg)
	}
}

func (c *WSGatewayClient) handleMessage(raw []byte) {
	var frame struct {
		Type    string          `json:"type"`
		ID      string          `json:"id"`
		Event   string          `json:"event"`
		Ok      bool            `json:"ok"`
		Payload json.RawMessage `json:"payload"`
		Error   *wsError        `json:"error"`
	}
	if err := json.Unmarshal(raw, &frame); err != nil {
		return
	}

	switch frame.Type {
	case "event":
		if frame.Event == "connect.challenge" {
			var payload struct {
				Nonce string `json:"nonce"`
			}
			json.Unmarshal(frame.Payload, &payload)
			c.sendConnect(payload.Nonce)
		} else if frame.Event == "agent" {
			c.handleAgentEvent(frame.Payload)
		}

	case "res":
		if frame.Ok {
			var hello struct {
				Type string `json:"type"`
			}
			json.Unmarshal(frame.Payload, &hello)
			if hello.Type == "hello-ok" {
				select {
				case <-c.ready:
				default:
					close(c.ready)
				}
			}
		}

		c.mu.Lock()
		p, ok := c.pending[frame.ID]
		if ok {
			delete(c.pending, frame.ID)
		}
		c.mu.Unlock()

		if ok {
			if frame.Ok {
				p.ch <- wsResponse{ok: true, payload: frame.Payload}
			} else {
				errMsg := "request failed"
				if frame.Error != nil {
					errMsg = frame.Error.Message
				}
				p.ch <- wsResponse{err: errMsg}
			}
		}
	}
}

func (c *WSGatewayClient) handleAgentEvent(payload json.RawMessage) {
	var ev struct {
		RunID  string `json:"runId"`
		Stream string `json:"stream"`
		Data   struct {
			Text  string `json:"text"`
			Phase string `json:"phase"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &ev); err != nil || ev.RunID == "" {
		return
	}

	c.mu.Lock()
	run, ok := c.runs[ev.RunID]
	c.mu.Unlock()
	if !ok {
		return
	}

	switch ev.Stream {
	case "assistant":
		// ev.Data.Text is the accumulated text so far.
		if ev.Data.Text != "" {
			c.mu.Lock()
			run.text = ev.Data.Text
			c.mu.Unlock()
		}
	case "lifecycle":
		if ev.Data.Phase == "end" {
			select {
			case <-run.done:
			default:
				close(run.done)
			}
		}
	}
}

func (c *WSGatewayClient) sendConnect(nonce string) {
	params := map[string]interface{}{
		"minProtocol": 3,
		"maxProtocol": 3,
		"client": map[string]interface{}{
			"id":       "gateway-client",
			"version":  "dev",
			"platform": "server",
			"mode":     "backend",
		},
		"role":   "operator",
		"scopes": []string{"operator.admin"},
		"caps":   []string{},
	}
	if c.token != "" {
		params["auth"] = map[string]interface{}{
			"token": c.token,
		}
	}

	c.seq++
	id := fmt.Sprintf("connect-%d", c.seq)
	frame := map[string]interface{}{
		"type":   "req",
		"id":     id,
		"method": "connect",
		"params": params,
	}

	ch := make(chan wsResponse, 1)
	c.mu.Lock()
	c.pending[id] = &wsPending{ch: ch}
	c.mu.Unlock()

	data, _ := json.Marshal(frame)
	c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *WSGatewayClient) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	c.seq++
	id := fmt.Sprintf("req-%d", c.seq)
	frame := map[string]interface{}{
		"type":   "req",
		"id":     id,
		"method": method,
		"params": params,
	}

	ch := make(chan wsResponse, 1)
	c.mu.Lock()
	c.pending[id] = &wsPending{ch: ch}
	c.mu.Unlock()

	data, _ := json.Marshal(frame)
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("ws write: %w", err)
	}

	select {
	case resp := <-ch:
		if resp.err != "" {
			return nil, fmt.Errorf("gateway rpc error: %s", resp.err)
		}
		return resp.payload, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	}
}

// SendMessage sends a message to the gateway via the "agent" RPC method.
// It collects the streamed response from agent events and returns when done.
func (c *WSGatewayClient) SendMessage(ctx context.Context, msg *types.Message) (*types.MessageResponse, error) {
	if c.conn == nil {
		if err := c.Connect(ctx); err != nil {
			return nil, fmt.Errorf("gateway connect: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	idemKey := "msg-" + msg.ID

	// Register the run tracker before sending so we don't miss events.
	run := &agentRun{done: make(chan struct{})}
	// We don't know the runId yet; register after we get the accepted response.

	params := map[string]interface{}{
		"message":        msg.Content,
		"idempotencyKey": idemKey,
		"agentId":        "main",
		"sessionKey":     "agent:main:claw-mesh:dashboard:" + msg.Source,
	}

	payload, err := c.call(ctx, "agent", params)
	if err != nil {
		return nil, err
	}

	var accepted struct {
		RunID  string `json:"runId"`
		Status string `json:"status"`
	}
	json.Unmarshal(payload, &accepted)
	if accepted.RunID == "" {
		return nil, fmt.Errorf("gateway agent: no runId in response")
	}

	// Register the run so handleAgentEvent can populate it.
	c.mu.Lock()
	c.runs[accepted.RunID] = run
	c.mu.Unlock()

	// Wait for the agent run to complete (lifecycle end event).
	select {
	case <-run.done:
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.runs, accepted.RunID)
		c.mu.Unlock()
		return nil, ctx.Err()
	}

	c.mu.Lock()
	response := run.text
	runErr := run.err
	delete(c.runs, accepted.RunID)
	c.mu.Unlock()

	if runErr != "" {
		return nil, fmt.Errorf("agent run error: %s", runErr)
	}
	if response == "" {
		response = "(no response)"
	}

	return &types.MessageResponse{
		MessageID: msg.ID,
		Response:  response,
	}, nil
}

// HealthCheck verifies the gateway is reachable via TCP.
func (c *WSGatewayClient) HealthCheck(_ context.Context) bool {
	conn, err := net.DialTimeout("tcp", c.endpoint, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Close closes the WebSocket connection.
func (c *WSGatewayClient) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

// ConnectAndLog connects to the gateway, logging the result.
func (c *WSGatewayClient) ConnectAndLog() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := c.Connect(ctx); err != nil {
		log.Printf("WARN: gateway ws connect failed: %v (will retry on first message)", err)
	} else {
		log.Printf("gateway ws connected to %s", c.endpoint)
	}
}
