package coordinator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
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

// backoff durations for retry attempts.
var retryBackoffs = []time.Duration{100 * time.Millisecond, 200 * time.Millisecond, 400 * time.Millisecond}

// ForwardMessage sends a message to the target node and returns the response.
// It retries on transient errors (502/503, network errors, connection reset, EOF)
// with exponential backoff.
func (f *Forwarder) ForwardMessage(ctx context.Context, node *types.Node, msg *types.Message, token string) (*types.MessageResponse, error) {
	maxAttempts := len(retryBackoffs) + 1
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := f.doForward(ctx, node, msg, token)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !isTransient(err) {
			return nil, err
		}
		if attempt < len(retryBackoffs) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryBackoffs[attempt]):
			}
		}
	}
	return nil, fmt.Errorf("forwarding failed after %d attempts: %w", maxAttempts, lastErr)
}

func (f *Forwarder) doForward(ctx context.Context, node *types.Node, msg *types.Message, token string) (*types.MessageResponse, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshaling message: %w", err)
	}

	url := fmt.Sprintf("http://%s/api/v1/messages", node.Endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("creating forward request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		// Network errors (temporary, connection reset, EOF) are transient.
		return nil, &transientError{cause: err, nodeID: node.ID}
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
	cause  error
}

func (e *transientError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("transient error forwarding to node %s: %v", e.nodeID, e.cause)
	}
	return fmt.Sprintf("node %s returned transient status %d", e.nodeID, e.status)
}

func (e *transientError) Unwrap() error { return e.cause }

func isTransient(err error) bool {
	var te *transientError
	if errors.As(err, &te) {
		return true
	}
	// Also catch raw network errors that slip through.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return false
}
