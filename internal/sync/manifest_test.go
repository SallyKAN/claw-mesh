package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

func TestHashBytes(t *testing.T) {
	h := HashBytes([]byte("hello world"))
	// SHA-256 of "hello world"
	want := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if h != want {
		t.Errorf("HashBytes = %s, want %s", h, want)
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.txt")
	os.WriteFile(p, []byte("hello world"), 0644)

	hash, size, err := HashFile(p)
	if err != nil {
		t.Fatalf("HashFile error: %v", err)
	}
	if size != 11 {
		t.Errorf("size = %d, want 11", size)
	}
	want := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != want {
		t.Errorf("hash = %s, want %s", hash, want)
	}
}

func TestHashFile_NotFound(t *testing.T) {
	_, _, err := HashFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestScanWorkspaceDir(t *testing.T) {
	dir := t.TempDir()
	// Create identity files.
	os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte("soul"), 0644)
	os.WriteFile(filepath.Join(dir, "IDENTITY.md"), []byte("identity"), 0644)
	os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("memory"), 0644)
	// Create memory subdir.
	os.MkdirAll(filepath.Join(dir, "memory"), 0755)
	os.WriteFile(filepath.Join(dir, "memory", "daily.md"), []byte("daily"), 0644)
	// Create skills subdir.
	os.MkdirAll(filepath.Join(dir, "skills", "coding"), 0755)
	os.WriteFile(filepath.Join(dir, "skills", "coding", "SKILL.md"), []byte("skill"), 0644)

	entries, err := ScanWorkspaceDir(dir)
	if err != nil {
		t.Fatalf("ScanWorkspaceDir error: %v", err)
	}

	paths := make(map[string]bool)
	for _, e := range entries {
		paths[e.Path] = true
	}

	expected := []string{"SOUL.md", "IDENTITY.md", "MEMORY.md", "memory/daily.md", "skills/coding/SKILL.md"}
	for _, p := range expected {
		if !paths[p] {
			t.Errorf("missing expected path: %s", p)
		}
	}
}

func TestScanWorkspaceDir_Empty(t *testing.T) {
	dir := t.TempDir()
	entries, err := ScanWorkspaceDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestDiffManifests_NewFiles(t *testing.T) {
	remote := types.SyncManifest{
		Files: []types.SyncFileEntry{
			{Path: "SOUL.md", SHA256: "aaa"},
			{Path: "NEW.md", SHA256: "bbb"},
		},
	}
	local := []types.SyncFileEntry{
		{Path: "SOUL.md", SHA256: "aaa"},
	}
	toDownload, toUpload := DiffManifests(remote, local)
	if len(toDownload) != 1 || toDownload[0].Path != "NEW.md" {
		t.Errorf("toDownload = %v, want [NEW.md]", toDownload)
	}
	if len(toUpload) != 0 {
		t.Errorf("toUpload = %v, want empty", toUpload)
	}
}

func TestDiffManifests_ModifiedFiles(t *testing.T) {
	remote := types.SyncManifest{
		Files: []types.SyncFileEntry{
			{Path: "SOUL.md", SHA256: "aaa"},
		},
	}
	local := []types.SyncFileEntry{
		{Path: "SOUL.md", SHA256: "bbb"},
	}
	toDownload, toUpload := DiffManifests(remote, local)
	if len(toDownload) != 1 {
		t.Errorf("toDownload = %d, want 1", len(toDownload))
	}
	if len(toUpload) != 1 {
		t.Errorf("toUpload = %d, want 1", len(toUpload))
	}
}

func TestDiffManifests_LocalOnly(t *testing.T) {
	remote := types.SyncManifest{}
	local := []types.SyncFileEntry{
		{Path: "LOCAL.md", SHA256: "ccc"},
	}
	toDownload, toUpload := DiffManifests(remote, local)
	if len(toDownload) != 0 {
		t.Errorf("toDownload = %d, want 0", len(toDownload))
	}
	if len(toUpload) != 1 || toUpload[0].Path != "LOCAL.md" {
		t.Errorf("toUpload = %v, want [LOCAL.md]", toUpload)
	}
}

func TestDiffManifests_NoChanges(t *testing.T) {
	remote := types.SyncManifest{
		Files: []types.SyncFileEntry{
			{Path: "SOUL.md", SHA256: "aaa"},
		},
	}
	local := []types.SyncFileEntry{
		{Path: "SOUL.md", SHA256: "aaa"},
	}
	toDownload, toUpload := DiffManifests(remote, local)
	if len(toDownload) != 0 || len(toUpload) != 0 {
		t.Errorf("expected no diff, got download=%d upload=%d", len(toDownload), len(toUpload))
	}
}

func TestManifestStore_ApplyPush(t *testing.T) {
	dir := t.TempDir()
	store, err := NewManifestStore(
		filepath.Join(dir, "manifest.json"),
		filepath.Join(dir, "data"),
	)
	if err != nil {
		t.Fatalf("NewManifestStore: %v", err)
	}

	req := types.SyncPushRequest{
		NodeID: "node-1",
		Files: []types.SyncPushFile{
			{Path: "SOUL.md", Content: "hello soul", SHA256: HashBytes([]byte("hello soul"))},
			{Path: "memory/daily.md", Content: "daily note", SHA256: HashBytes([]byte("daily note"))},
		},
	}

	resp, err := store.ApplyPush(req)
	if err != nil {
		t.Fatalf("ApplyPush: %v", err)
	}
	if len(resp.Accepted) != 2 {
		t.Errorf("accepted = %d, want 2", len(resp.Accepted))
	}
	if resp.Version != 1 {
		t.Errorf("version = %d, want 1", resp.Version)
	}

	// Verify file was written.
	data, err := store.GetFileContent("SOUL.md")
	if err != nil {
		t.Fatalf("GetFileContent: %v", err)
	}
	if string(data) != "hello soul" {
		t.Errorf("content = %q, want %q", string(data), "hello soul")
	}
}

func TestManifestStore_ApplyPush_Conflict(t *testing.T) {
	dir := t.TempDir()
	store, err := NewManifestStore(
		filepath.Join(dir, "manifest.json"),
		filepath.Join(dir, "data"),
	)
	if err != nil {
		t.Fatalf("NewManifestStore: %v", err)
	}

	// First push.
	store.ApplyPush(types.SyncPushRequest{
		NodeID: "node-1",
		Files: []types.SyncPushFile{
			{Path: "SOUL.md", Content: "version 1", SHA256: HashBytes([]byte("version 1"))},
		},
	})

	// Second push with different content from different node.
	resp, _ := store.ApplyPush(types.SyncPushRequest{
		NodeID: "node-2",
		Files: []types.SyncPushFile{
			{Path: "SOUL.md", Content: "version 2", SHA256: HashBytes([]byte("version 2"))},
		},
	})

	if len(resp.Conflicts) != 1 {
		t.Errorf("conflicts = %d, want 1", len(resp.Conflicts))
	}
	if len(resp.Conflicts) > 0 && resp.Conflicts[0].Resolution != "kept_both" {
		t.Errorf("resolution = %s, want kept_both", resp.Conflicts[0].Resolution)
	}
}

func TestManifestStore_ApplyPush_Delete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewManifestStore(
		filepath.Join(dir, "manifest.json"),
		filepath.Join(dir, "data"),
	)
	if err != nil {
		t.Fatalf("NewManifestStore: %v", err)
	}

	// Push a file.
	store.ApplyPush(types.SyncPushRequest{
		NodeID: "node-1",
		Files:  []types.SyncPushFile{{Path: "SOUL.md", Content: "soul", SHA256: HashBytes([]byte("soul"))}},
	})

	// Delete it.
	resp, _ := store.ApplyPush(types.SyncPushRequest{
		NodeID: "node-1",
		Files:  []types.SyncPushFile{{Path: "SOUL.md", Delete: true}},
	})

	if len(resp.Accepted) != 1 {
		t.Errorf("accepted = %d, want 1", len(resp.Accepted))
	}
	m := store.GetManifest()
	if len(m.Files) != 0 {
		t.Errorf("manifest files = %d, want 0", len(m.Files))
	}
}

func TestManifestStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "manifest.json")
	dataDir := filepath.Join(dir, "data")

	store1, _ := NewManifestStore(storePath, dataDir)
	store1.ApplyPush(types.SyncPushRequest{
		NodeID: "node-1",
		Files:  []types.SyncPushFile{{Path: "SOUL.md", Content: "persist", SHA256: HashBytes([]byte("persist"))}},
	})

	// Create new store from same path — should load persisted data.
	store2, err := NewManifestStore(storePath, dataDir)
	if err != nil {
		t.Fatalf("NewManifestStore reload: %v", err)
	}
	m := store2.GetManifest()
	if len(m.Files) != 1 {
		t.Fatalf("reloaded files = %d, want 1", len(m.Files))
	}
	if m.Files[0].Path != "SOUL.md" {
		t.Errorf("path = %s, want SOUL.md", m.Files[0].Path)
	}
}

func TestManifestStore_SeedFromWorkspace(t *testing.T) {
	wsDir := t.TempDir()
	os.WriteFile(filepath.Join(wsDir, "SOUL.md"), []byte("soul content"), 0644)
	os.MkdirAll(filepath.Join(wsDir, "memory"), 0755)
	os.WriteFile(filepath.Join(wsDir, "memory", "notes.md"), []byte("notes"), 0644)

	dir := t.TempDir()
	store, _ := NewManifestStore(
		filepath.Join(dir, "manifest.json"),
		filepath.Join(dir, "data"),
	)

	if err := store.SeedFromWorkspace(wsDir); err != nil {
		t.Fatalf("SeedFromWorkspace: %v", err)
	}

	m := store.GetManifest()
	if len(m.Files) != 2 {
		t.Errorf("files = %d, want 2", len(m.Files))
	}

	// Verify file was copied to data dir.
	data, err := store.GetFileContent("SOUL.md")
	if err != nil {
		t.Fatalf("GetFileContent: %v", err)
	}
	if string(data) != "soul content" {
		t.Errorf("content = %q, want %q", string(data), "soul content")
	}
}

func TestManifestStore_GetFileContent_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewManifestStore(
		filepath.Join(dir, "manifest.json"),
		filepath.Join(dir, "data"),
	)
	_, err := store.GetFileContent("../../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal")
	}
}

func TestManifestStore_GetNodeSyncStatuses(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewManifestStore(
		filepath.Join(dir, "manifest.json"),
		filepath.Join(dir, "data"),
	)

	store.RecordNodeSync("node-1", 5, 3)
	store.RecordNodeSync("node-2", 5, 2)

	statuses := store.GetNodeSyncStatuses()
	if len(statuses) != 2 {
		t.Errorf("statuses = %d, want 2", len(statuses))
	}
}

// Suppress unused import warning for time package.
var _ = time.Now
