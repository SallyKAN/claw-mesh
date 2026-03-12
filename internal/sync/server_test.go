package sync

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

func setupTestSyncServer(t *testing.T) (*SyncServer, *ManifestStore) {
	t.Helper()
	dir := t.TempDir()
	store, err := NewManifestStore(
		filepath.Join(dir, "manifest.json"),
		filepath.Join(dir, "data"),
	)
	if err != nil {
		t.Fatalf("NewManifestStore: %v", err)
	}
	// Seed some data.
	os.MkdirAll(filepath.Join(dir, "data"), 0700)
	os.WriteFile(filepath.Join(dir, "data", "SOUL.md"), []byte("test soul"), 0644)
	store.ApplyPush(types.SyncPushRequest{
		NodeID: "seed",
		Files: []types.SyncPushFile{
			{Path: "SOUL.md", Content: "test soul", SHA256: HashBytes([]byte("test soul"))},
		},
	})
	return NewSyncServer(store), store
}

func TestHandleGetManifest(t *testing.T) {
	srv, _ := setupTestSyncServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/manifest", nil)
	rr := httptest.NewRecorder()
	srv.HandleGetManifest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var m types.SyncManifest
	if err := json.NewDecoder(rr.Body).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(m.Files) != 1 {
		t.Errorf("files = %d, want 1", len(m.Files))
	}
}

func TestHandleGetFile(t *testing.T) {
	srv, _ := setupTestSyncServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/file?path=SOUL.md", nil)
	rr := httptest.NewRecorder()
	srv.HandleGetFile(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if rr.Body.String() != "test soul" {
		t.Errorf("body = %q, want %q", rr.Body.String(), "test soul")
	}
}

func TestHandleGetFile_NotFound(t *testing.T) {
	srv, _ := setupTestSyncServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/file?path=NOPE.md", nil)
	rr := httptest.NewRecorder()
	srv.HandleGetFile(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestHandleGetFile_MissingParam(t *testing.T) {
	srv, _ := setupTestSyncServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/file", nil)
	rr := httptest.NewRecorder()
	srv.HandleGetFile(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rr.Code)
	}
}

func TestHandlePush(t *testing.T) {
	srv, _ := setupTestSyncServer(t)
	pushReq := types.SyncPushRequest{
		NodeID: "node-test",
		Files: []types.SyncPushFile{
			{Path: "IDENTITY.md", Content: "new identity", SHA256: HashBytes([]byte("new identity"))},
		},
	}
	body, _ := json.Marshal(pushReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/push", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.HandlePush(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	var resp types.SyncPushResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Accepted) != 1 {
		t.Errorf("accepted = %d, want 1", len(resp.Accepted))
	}
}

func TestHandleStatus(t *testing.T) {
	srv, store := setupTestSyncServer(t)
	store.RecordNodeSync("node-1", 1, 3)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/status", nil)
	rr := httptest.NewRecorder()
	srv.HandleStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var statuses []types.SyncNodeStatus
	if err := json.NewDecoder(rr.Body).Decode(&statuses); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// setupTestSyncServer pushes as "seed", plus we added "node-1".
	if len(statuses) != 2 {
		t.Errorf("statuses = %d, want 2", len(statuses))
	}
}

func TestHandleStatus_Empty(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewManifestStore(
		filepath.Join(dir, "manifest.json"),
		filepath.Join(dir, "data"),
	)
	srv := NewSyncServer(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/status", nil)
	rr := httptest.NewRecorder()
	srv.HandleStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var statuses []types.SyncNodeStatus
	json.NewDecoder(rr.Body).Decode(&statuses)
	if len(statuses) != 0 {
		t.Errorf("statuses = %d, want 0", len(statuses))
	}
}
