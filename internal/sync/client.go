package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

// SyncClient handles file sync from the node side.
type SyncClient struct {
	coordinatorURL string
	token          string
	nodeID         string
	workspaceDir   string
	httpClient     *http.Client
	lastVersion    int64
}

// NewSyncClient creates a sync client for a node.
func NewSyncClient(coordinatorURL, token, nodeID, workspaceDir string) *SyncClient {
	return &SyncClient{
		coordinatorURL: coordinatorURL,
		token:          token,
		nodeID:         nodeID,
		workspaceDir:   workspaceDir,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// PullSync fetches the remote manifest, diffs against local, and downloads changed files.
func (c *SyncClient) PullSync() error {
	// 1. Get remote manifest.
	manifest, err := c.fetchManifest()
	if err != nil {
		return fmt.Errorf("fetch manifest: %w", err)
	}

	if manifest.Version == c.lastVersion {
		return nil // no changes
	}

	// 2. Scan local workspace.
	local, err := ScanWorkspaceDir(c.workspaceDir)
	if err != nil {
		return fmt.Errorf("scan local: %w", err)
	}

	// 3. Diff.
	toDownload, _ := DiffManifests(manifest, local)
	if len(toDownload) == 0 {
		c.lastVersion = manifest.Version
		return nil
	}

	// 4. Download each file.
	downloaded := 0
	for _, f := range toDownload {
		data, err := c.fetchFile(f.Path)
		if err != nil {
			log.Printf("sync/pull: failed to download %s: %v", f.Path, err)
			continue
		}
		target := filepath.Join(c.workspaceDir, f.Path)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			log.Printf("sync/pull: failed to create dir for %s: %v", f.Path, err)
			continue
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			log.Printf("sync/pull: failed to write %s: %v", f.Path, err)
			continue
		}
		downloaded++
	}

	c.lastVersion = manifest.Version
	if downloaded > 0 {
		log.Printf("sync/pull: downloaded %d files (manifest v%d)", downloaded, manifest.Version)
	}
	return nil
}

// PushSync scans local workspace for changes and uploads them to the coordinator.
func (c *SyncClient) PushSync() error {
	// 1. Scan local workspace.
	local, err := ScanWorkspaceDir(c.workspaceDir)
	if err != nil {
		return fmt.Errorf("scan local: %w", err)
	}

	// 2. Get remote manifest for diff.
	manifest, err := c.fetchManifest()
	if err != nil {
		return fmt.Errorf("fetch manifest: %w", err)
	}

	// 3. Diff — find files to upload.
	_, toUpload := DiffManifests(manifest, local)
	if len(toUpload) == 0 {
		return nil
	}

	// 4. Build push request.
	req := types.SyncPushRequest{
		NodeID: c.nodeID,
	}
	for _, f := range toUpload {
		data, err := os.ReadFile(filepath.Join(c.workspaceDir, f.Path))
		if err != nil {
			continue
		}
		req.Files = append(req.Files, types.SyncPushFile{
			Path:    f.Path,
			Content: string(data),
			SHA256:  HashBytes(data),
		})
	}

	if len(req.Files) == 0 {
		return nil
	}

	// 5. POST push.
	resp, err := c.pushFiles(req)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}

	c.lastVersion = resp.Version
	log.Printf("sync/push: uploaded %d files, %d conflicts (manifest v%d)",
		len(resp.Accepted), len(resp.Conflicts), resp.Version)
	return nil
}

// SyncOnce performs a full sync cycle: push local changes, then pull remote changes.
func (c *SyncClient) SyncOnce() error {
	if err := c.PushSync(); err != nil {
		log.Printf("sync: push failed: %v", err)
	}
	return c.PullSync()
}

func (c *SyncClient) fetchManifest() (types.SyncManifest, error) {
	var m types.SyncManifest
	data, err := c.doGet(c.coordinatorURL + "/api/v1/sync/manifest")
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("parse manifest: %w", err)
	}
	return m, nil
}

func (c *SyncClient) fetchFile(path string) ([]byte, error) {
	return c.doGet(c.coordinatorURL + "/api/v1/sync/file?path=" + path)
}

func (c *SyncClient) pushFiles(req types.SyncPushRequest) (*types.SyncPushResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.coordinatorURL+"/api/v1/sync/push", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}

	var pushResp types.SyncPushResponse
	if err := json.Unmarshal(data, &pushResp); err != nil {
		return nil, fmt.Errorf("parse push response: %w", err)
	}
	return &pushResp, nil
}

func (c *SyncClient) doGet(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}
