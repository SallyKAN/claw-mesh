package coordinator

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/SallyKAN/claw-mesh/internal/config"
)

func TestHandleSeedConfig(t *testing.T) {
	// Create a temp openclaw.json with mixed keys.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "openclaw.json")
	raw := map[string]any{
		"auth":     map[string]any{"profiles": []string{"default"}},
		"models":   map[string]any{"providers": []string{"anthropic"}},
		"env":      map[string]any{"ANTHROPIC_API_KEY": "sk-test"},
		"channels": map[string]any{"telegram": map[string]any{"token": "tg-123"}},
		"gateway":  map[string]any{"port": 18789},
		"meta":     map[string]any{"lastTouchedAt": "2026-03-01"},
		"wizard":   map[string]any{"lastRunAt": "2026-03-01"},
		"bindings": []string{"b1"},
	}
	data, _ := json.Marshal(raw)
	os.WriteFile(cfgPath, data, 0644)

	srv := &Server{cfg: &config.CoordinatorConfig{OpenClawConfig: cfgPath}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/seed/config", nil)
	rr := httptest.NewRecorder()
	srv.handleSeedConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	// Should include shared keys.
	for _, key := range []string{"auth", "models", "env"} {
		if _, ok := result[key]; !ok {
			t.Errorf("expected key %q in seed config", key)
		}
	}

	// Should exclude local keys.
	for _, key := range []string{"channels", "gateway", "meta", "wizard", "bindings"} {
		if _, ok := result[key]; ok {
			t.Errorf("key %q should be excluded from seed config", key)
		}
	}
}

func TestHandleSeedConfig_NotFound(t *testing.T) {
	// Use a non-existent path AND override home to prevent auto-detect fallback.
	t.Setenv("HOME", t.TempDir())
	srv := &Server{cfg: &config.CoordinatorConfig{OpenClawConfig: "/nonexistent/path.json"}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/seed/config", nil)
	rr := httptest.NewRecorder()
	srv.handleSeedConfig(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestHandleSeedWorkspace(t *testing.T) {
	// Create a temp workspace with identity + memory files.
	wsDir := t.TempDir()
	os.WriteFile(filepath.Join(wsDir, "SOUL.md"), []byte("# Soul"), 0644)
	os.WriteFile(filepath.Join(wsDir, "IDENTITY.md"), []byte("# Identity"), 0644)
	os.WriteFile(filepath.Join(wsDir, "AGENTS.md"), []byte("# Agents"), 0644)
	os.WriteFile(filepath.Join(wsDir, "MEMORY.md"), []byte("# Memory"), 0644)
	// Should NOT be included.
	os.WriteFile(filepath.Join(wsDir, "TOOLS.md"), []byte("# Tools"), 0644)

	memDir := filepath.Join(wsDir, "memory")
	os.MkdirAll(memDir, 0755)
	os.WriteFile(filepath.Join(memDir, "2026-03-01.md"), []byte("# Day 1"), 0644)
	os.WriteFile(filepath.Join(memDir, "2026-03-02.md"), []byte("# Day 2"), 0644)

	srv := &Server{cfg: &config.CoordinatorConfig{WorkspaceDir: wsDir}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/seed/workspace", nil)
	rr := httptest.NewRecorder()
	srv.handleSeedWorkspace(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp WorkspaceSeedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Should have 4 identity files + 2 memory files = 6.
	if len(resp.Files) != 6 {
		names := make([]string, len(resp.Files))
		for i, f := range resp.Files {
			names[i] = f.Path
		}
		t.Fatalf("expected 6 files, got %d: %v", len(resp.Files), names)
	}

	// TOOLS.md should not be present.
	for _, f := range resp.Files {
		if f.Path == "TOOLS.md" {
			t.Error("TOOLS.md should not be in seed workspace")
		}
	}
}

func TestHandleSeedWorkspace_NotConfigured(t *testing.T) {
	// Override home to prevent auto-detect fallback.
	t.Setenv("HOME", t.TempDir())
	srv := &Server{cfg: &config.CoordinatorConfig{WorkspaceDir: "/nonexistent/dir"}}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/seed/workspace", nil)
	rr := httptest.NewRecorder()
	srv.handleSeedWorkspace(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}
