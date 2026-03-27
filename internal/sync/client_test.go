package sync

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

func TestSyncClient_PullSync(t *testing.T) {
	// Set up a mock coordinator server.
	manifest := types.SyncManifest{
		Version: 1,
		Files: []types.SyncFileEntry{
			{Path: "SOUL.md", SHA256: HashBytes([]byte("remote soul"))},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/sync/manifest":
			json.NewEncoder(w).Encode(manifest)
		case "/api/v1/sync/file":
			path := r.URL.Query().Get("path")
			if path == "SOUL.md" {
				w.Write([]byte("remote soul"))
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	wsDir := t.TempDir()
	client := NewSyncClient(srv.URL, "token", "node-1", wsDir)

	if err := client.PullSync(); err != nil {
		t.Fatalf("PullSync: %v", err)
	}

	// Verify file was downloaded.
	data, err := os.ReadFile(filepath.Join(wsDir, "SOUL.md"))
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(data) != "remote soul" {
		t.Errorf("content = %q, want %q", string(data), "remote soul")
	}
}

func TestSyncClient_PullSync_NoChanges(t *testing.T) {
	manifest := types.SyncManifest{Version: 1}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(manifest)
	}))
	defer srv.Close()

	wsDir := t.TempDir()
	client := NewSyncClient(srv.URL, "", "node-1", wsDir)
	client.lastVersion = 1 // Already at version 1.

	if err := client.PullSync(); err != nil {
		t.Fatalf("PullSync: %v", err)
	}
}

func TestSyncClient_PushSync(t *testing.T) {
	var pushed types.SyncPushRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/sync/manifest":
			json.NewEncoder(w).Encode(types.SyncManifest{Version: 0})
		case "/api/v1/sync/push":
			json.NewDecoder(r.Body).Decode(&pushed)
			json.NewEncoder(w).Encode(types.SyncPushResponse{
				Accepted: []string{"SOUL.md"},
				Version:  1,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	wsDir := t.TempDir()
	os.WriteFile(filepath.Join(wsDir, "SOUL.md"), []byte("local soul"), 0644)

	client := NewSyncClient(srv.URL, "token", "node-1", wsDir)
	if err := client.PushSync(); err != nil {
		t.Fatalf("PushSync: %v", err)
	}

	if pushed.NodeID != "node-1" {
		t.Errorf("nodeID = %s, want node-1", pushed.NodeID)
	}
	if len(pushed.Files) != 1 {
		t.Errorf("files = %d, want 1", len(pushed.Files))
	}
}

// --- Watcher tests ---

func TestWatcher_DetectsChanges(t *testing.T) {
	wsDir := t.TempDir()
	os.WriteFile(filepath.Join(wsDir, "SOUL.md"), []byte("initial"), 0644)

	var callCount int32
	w := NewWatcher(wsDir, 100*time.Millisecond, func() {
		atomic.AddInt32(&callCount, 1)
	})
	w.Start()
	defer w.Stop()

	// Wait for initial snapshot.
	time.Sleep(50 * time.Millisecond)

	// Modify file.
	os.WriteFile(filepath.Join(wsDir, "SOUL.md"), []byte("modified"), 0644)

	// Wait for watcher to detect.
	time.Sleep(250 * time.Millisecond)

	if c := atomic.LoadInt32(&callCount); c < 1 {
		t.Errorf("onChange called %d times, want >= 1", c)
	}
}

func TestWatcher_NoChangeNoCallback(t *testing.T) {
	wsDir := t.TempDir()
	os.WriteFile(filepath.Join(wsDir, "SOUL.md"), []byte("stable"), 0644)

	var callCount int32
	w := NewWatcher(wsDir, 100*time.Millisecond, func() {
		atomic.AddInt32(&callCount, 1)
	})
	w.Start()
	defer w.Stop()

	// Wait a few cycles.
	time.Sleep(350 * time.Millisecond)

	if c := atomic.LoadInt32(&callCount); c != 0 {
		t.Errorf("onChange called %d times, want 0", c)
	}
}

func TestWatcher_Stop(t *testing.T) {
	wsDir := t.TempDir()
	w := NewWatcher(wsDir, 50*time.Millisecond, func() {})
	w.Start()
	w.Stop()
	// Double stop should not panic.
	w.Stop()
}
