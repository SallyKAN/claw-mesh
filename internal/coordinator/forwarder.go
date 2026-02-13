package coordinator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/snapek/claw-mesh/internal/types"
)

// Forwarder sends messages to node endpoints via HTTP.
type Forwarder struct {
	client *http.Client
}

// NewForwarder creates a message forwarder with sensible defaults.
func NewForwarder() *Forwarder {
	return &Forwarder{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// ForwardMessage sends a message to the target node and returns the response.
// It retries once on transient errors (502/503).
func (f *Forwarder) ForwardMessage(node *types.Node, msg *types.Message, token string) (*types.MessageResponse, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		resp, err := f.doForward(node, msg, token)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		// Only retry on transient errors.
		if !isTransient(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("forwarding failed after retry: %w", lastErr)
}

func (f *Forwarder) doForward(node *types.Node, msg *types.Message, token string) (*types.MessageResponse, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshaling message: %w", err)
	}

	url := fmt.Sprintf("http://%s/api/v1/messages", node.Endpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating forward request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("forwarding to node %s: %w", node.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusServiceUnavailable {
		return nil, &transientError{status: resp.StatusCode, nodeID: node.ID}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("node %s returned status %d: %s", node.ID, resp.StatusCode, string(body))
	}

	var msgResp types.MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
		return nil, fmt.Errorf("decoding response from node %s: %w", node.ID, err)
	}
	return &msgResp, nil
}

// transientError represents a retryable forwarding failure.
type transientError struct {
	status int
	nodeID string
}

func (e *transientError) Error() string {
	return fmt.Sprintf("node %s returned transient status %d", e.nodeID, e.status)
}

func isTransient(err error) bool {
	_, ok := err.(*transientError)
	return ok
}
