package node

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

func TestHTTPGatewayClient_SendMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected /v1/chat/completions, got %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}

		var req types.ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decoding request: %v", err)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "hello" {
			t.Errorf("unexpected messages: %+v", req.Messages)
		}

		json.NewEncoder(w).Encode(types.ChatCompletionResponse{
			ID: "chatcmpl-123",
			Choices: []types.Choice{{
				Message:      types.ChatMessage{Role: "assistant", Content: "hi there"},
				FinishReason: "stop",
			}},
		})
	}))
	defer srv.Close()

	// Strip http:// prefix since HTTPGatewayClient adds it.
	endpoint := srv.Listener.Addr().String()
	client := NewHTTPGatewayClient(endpoint, "test-token", 30)

	resp, err := client.SendMessage(context.Background(), &types.Message{
		ID:      "msg-1",
		Content: "hello",
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if resp.MessageID != "msg-1" {
		t.Errorf("expected message_id msg-1, got %s", resp.MessageID)
	}
	if resp.Response != "hi there" {
		t.Errorf("expected response 'hi there', got %s", resp.Response)
	}
}

func TestHTTPGatewayClient_SendMessage_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer srv.Close()

	client := NewHTTPGatewayClient(srv.Listener.Addr().String(), "bad-token", 30)
	_, err := client.SendMessage(context.Background(), &types.Message{
		ID:      "msg-2",
		Content: "test",
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestHTTPGatewayClient_SendMessage_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(types.ChatCompletionResponse{
			ID:      "chatcmpl-empty",
			Choices: []types.Choice{},
		})
	}))
	defer srv.Close()

	client := NewHTTPGatewayClient(srv.Listener.Addr().String(), "", 30)
	resp, err := client.SendMessage(context.Background(), &types.Message{
		ID:      "msg-3",
		Content: "test",
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if resp.Response != "" {
		t.Errorf("expected empty response, got %s", resp.Response)
	}
}

func TestHTTPGatewayClient_HealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewHTTPGatewayClient(srv.Listener.Addr().String(), "", 30)
	if !client.HealthCheck(context.Background()) {
		t.Error("expected health check to pass")
	}

	// Unreachable endpoint.
	badClient := NewHTTPGatewayClient("127.0.0.1:1", "", 30)
	if badClient.HealthCheck(context.Background()) {
		t.Error("expected health check to fail for unreachable endpoint")
	}
}

func TestResolveGatewayToken(t *testing.T) {
	// CLI flag takes precedence.
	if got := ResolveGatewayToken("cli-token", "discovered"); got != "cli-token" {
		t.Errorf("expected cli-token, got %s", got)
	}

	// Env var takes precedence over discovered.
	os.Setenv("OPENCLAW_GATEWAY_TOKEN", "env-token")
	defer os.Unsetenv("OPENCLAW_GATEWAY_TOKEN")
	if got := ResolveGatewayToken("", "discovered"); got != "env-token" {
		t.Errorf("expected env-token, got %s", got)
	}

	// Discovered token as fallback.
	os.Unsetenv("OPENCLAW_GATEWAY_TOKEN")
	if got := ResolveGatewayToken("", "discovered"); got != "discovered" {
		t.Errorf("expected discovered, got %s", got)
	}
}
