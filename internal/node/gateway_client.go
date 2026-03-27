package node

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

// GatewayClient is the interface for communicating with a local OpenClaw Gateway.
type GatewayClient interface {
	// SendMessage forwards a claw-mesh message to the gateway and returns the response.
	SendMessage(ctx context.Context, msg *types.Message) (*types.MessageResponse, error)
	// StreamMessage forwards a message to the gateway and writes the streaming response
	// as SSE events to w. The caller must set SSE response headers before calling this.
	StreamMessage(ctx context.Context, msg *types.Message, w http.ResponseWriter) error
	// HealthCheck returns true if the gateway is reachable and responsive.
	HealthCheck(ctx context.Context) bool
	// Close releases any resources held by the client.
	Close() error
}

// HTTPGatewayClient talks to an OpenClaw Gateway via the /v1/chat/completions HTTP API.
type HTTPGatewayClient struct {
	endpoint string
	token    string
	client   *http.Client
}

// NewHTTPGatewayClient creates a client for the OpenClaw Gateway HTTP API.
func NewHTTPGatewayClient(endpoint, token string, timeoutSec int) *HTTPGatewayClient {
	if timeoutSec <= 0 {
		timeoutSec = 120
	}
	return &HTTPGatewayClient{
		endpoint: endpoint,
		token:    token,
		client:   &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
	}
}

// SendMessage converts a claw-mesh Message to an OpenAI ChatCompletion request,
// sends it to the gateway, and maps the response back.
func (c *HTTPGatewayClient) SendMessage(ctx context.Context, msg *types.Message) (*types.MessageResponse, error) {
	reqBody := types.ChatCompletionRequest{
		Model: "default",
		Messages: []types.ChatMessage{
			{Role: "user", Content: msg.Content},
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling completion request: %w", err)
	}

	url := "http://" + c.endpoint + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating gateway request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gateway request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading gateway response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("gateway auth failed (401): %s", string(body))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gateway returned %d: %s", resp.StatusCode, string(body))
	}

	var completionResp types.ChatCompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		return nil, fmt.Errorf("decoding gateway response: %w", err)
	}

	content := ""
	if len(completionResp.Choices) > 0 {
		content = completionResp.Choices[0].Message.Content
	}

	return &types.MessageResponse{
		MessageID: msg.ID,
		Response:  content,
	}, nil
}

// StreamMessage sends a streaming chat completion request to the gateway and writes
// SSE events (data: {"type":"delta","delta":"..."}\n\n) to w.
// The caller must set SSE response headers before calling this.
func (c *HTTPGatewayClient) StreamMessage(ctx context.Context, msg *types.Message, w http.ResponseWriter) error {
	reqBody := types.ChatCompletionRequest{
		Model:    "default",
		Messages: []types.ChatMessage{{Role: "user", Content: msg.Content}},
		Stream:   true,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling stream request: %w", err)
	}

	url := "http://" + c.endpoint + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating gateway stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("gateway stream request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("gateway returned %d: %s", resp.StatusCode, string(body))
	}

	flusher, _ := w.(http.Flusher)
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			if flusher != nil {
				flusher.Flush()
			}
			return nil
		}
		var chunk types.ChatCompletionStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		sc := types.StreamChunk{Type: "delta", Delta: delta}
		b, _ := json.Marshal(sc)
		fmt.Fprintf(w, "data: %s\n\n", b)
		if flusher != nil {
			flusher.Flush()
		}
	}
	return scanner.Err()
}

// HealthCheck verifies the gateway is reachable via TCP.
func (c *HTTPGatewayClient) HealthCheck(_ context.Context) bool {
	conn, err := net.DialTimeout("tcp", c.endpoint, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Close is a no-op for the HTTP client.
func (c *HTTPGatewayClient) Close() error {
	return nil
}

// ResolveGatewayToken returns the gateway auth token using precedence:
// CLI flag > OPENCLAW_GATEWAY_TOKEN env > CLAWDBOT_GATEWAY_TOKEN env > discovered token.
func ResolveGatewayToken(cliToken, discoveredToken string) string {
	if cliToken != "" {
		return cliToken
	}
	if t := os.Getenv("OPENCLAW_GATEWAY_TOKEN"); t != "" {
		return t
	}
	if t := os.Getenv("CLAWDBOT_GATEWAY_TOKEN"); t != "" {
		return t
	}
	return discoveredToken
}
