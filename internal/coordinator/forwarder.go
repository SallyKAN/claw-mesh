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
	"strings"
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
		client: &http.Client{Timeout: 180 * time.Second},
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

	url := f.nodeBaseURL(node) + "/api/v1/messages"
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

// ForwardMessageAsync sends a message to the node's async endpoint, then polls
// for status until completion. The onPartial callback is called whenever a new
// partial response is available. The context controls the overall timeout.
func (f *Forwarder) ForwardMessageAsync(ctx context.Context, node *types.Node, msg *types.Message, token string, onPartial func(string)) (*types.MessageResponse, error) {
	// Step 1: POST to async endpoint — should return quickly.
	accepted, err := f.doForwardAsync(ctx, node, msg, token)
	if err != nil {
		return nil, fmt.Errorf("async submit to node %s: %w", node.ID, err)
	}

	// Step 2: Poll for status.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := f.pollMessageStatus(ctx, node, accepted.MessageID, token)
			if err != nil {
				// Transient poll errors — keep trying.
				continue
			}

			if status.PartialResponse != "" && onPartial != nil {
				onPartial(status.PartialResponse)
			}

			switch status.Status {
			case "completed":
				return &types.MessageResponse{
					MessageID: status.MessageID,
					NodeID:    node.ID,
					Response:  status.Response,
				}, nil
			case "failed":
				errMsg := status.Error
				if errMsg == "" {
					errMsg = "node processing failed"
				}
				return nil, fmt.Errorf("node %s: %s", node.ID, errMsg)
			}
			// accepted / processing — keep polling.
		}
	}
}

func (f *Forwarder) doForwardAsync(ctx context.Context, node *types.Node, msg *types.Message, token string) (*types.NodeAsyncAccepted, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshaling message: %w", err)
	}

	base := f.nodeBaseURL(node)
	url := base + "/api/v1/messages/async"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("node returned status %d: %s", resp.StatusCode, string(body))
	}

	var accepted types.NodeAsyncAccepted
	if err := json.NewDecoder(resp.Body).Decode(&accepted); err != nil {
		return nil, fmt.Errorf("decoding accepted response: %w", err)
	}
	return &accepted, nil
}

func (f *Forwarder) pollMessageStatus(ctx context.Context, node *types.Node, msgID, token string) (*types.NodeMessageStatus, error) {
	base := f.nodeBaseURL(node)
	url := base + "/api/v1/messages/" + msgID + "/status"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status poll returned %d", resp.StatusCode)
	}

	var status types.NodeMessageStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (f *Forwarder) nodeBaseURL(node *types.Node) string {
	if strings.HasPrefix(node.Endpoint, "http://") || strings.HasPrefix(node.Endpoint, "https://") {
		return strings.TrimRight(node.Endpoint, "/")
	}
	return "http://" + node.Endpoint
}
