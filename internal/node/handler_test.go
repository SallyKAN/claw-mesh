package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

// mockGatewayClient implements GatewayClient for testing.
type mockGatewayClient struct {
	response *types.MessageResponse
	err      error
	healthy  bool
}

func (m *mockGatewayClient) SendMessage(_ context.Context, msg *types.Message) (*types.MessageResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	resp := *m.response
	resp.MessageID = msg.ID
	return &resp, nil
}

func (m *mockGatewayClient) HealthCheck(_ context.Context) bool { return m.healthy }
func (m *mockGatewayClient) Close() error                      { return nil }

func postMessage(handler http.Handler, msg types.Message) *httptest.ResponseRecorder {
	body, _ := json.Marshal(msg)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestHandler_WithGateway(t *testing.T) {
	mock := &mockGatewayClient{
		response: &types.MessageResponse{Response: "AI says hello"},
		healthy:  true,
	}
	h := NewHandler(nil, mock)

	rr := postMessage(h, types.Message{ID: "msg-1", Content: "hello"})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp types.MessageResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.MessageID != "msg-1" {
		t.Errorf("expected message_id msg-1, got %s", resp.MessageID)
	}
	if resp.Response != "AI says hello" {
		t.Errorf("expected 'AI says hello', got %s", resp.Response)
	}
}

func TestHandler_WithoutGateway_Fallback(t *testing.T) {
	h := NewHandler(nil, nil)

	rr := postMessage(h, types.Message{ID: "msg-2", Content: "test message"})

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp types.MessageResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.MessageID != "msg-2" {
		t.Errorf("expected message_id msg-2, got %s", resp.MessageID)
	}
	// Fallback should include the original message content.
	if resp.Response == "test message" {
		t.Error("expected fallback response, not raw echo")
	}
}

func TestHandler_GatewayError(t *testing.T) {
	mock := &mockGatewayClient{
		err:     fmt.Errorf("connection refused"),
		healthy: false,
	}
	h := NewHandler(nil, mock)

	rr := postMessage(h, types.Message{ID: "msg-3", Content: "hello"})

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandler_InvalidBody(t *testing.T) {
	h := NewHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandler_MissingFields(t *testing.T) {
	h := NewHandler(nil, nil)

	rr := postMessage(h, types.Message{ID: "", Content: "no id"})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing id, got %d", rr.Code)
	}

	rr = postMessage(h, types.Message{ID: "msg-4", Content: ""})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing content, got %d", rr.Code)
	}
}
