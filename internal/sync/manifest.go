package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

// syncablePatterns defines which files are eligible for sync.
// Identity files, memory files, and skills.
var syncablePatterns = []string{
	"SOUL.md",
	"IDENTITY.md",
	"USER.md",
	"AGENTS.md",
	"MEMORY.md",
}

// ManifestStore manages the sync manifest and file storage on the coordinator.
type ManifestStore struct {
	mu        sync.RWMutex
	storePath string // path to sync-manifest.json
	dataDir   string // path to sync-data/
	manifest  types.SyncManifest
	nodeSync  map[string]*types.SyncNodeStatus // nodeID -> status
}

// NewManifestStore creates a new manifest store.
// storePath is the JSON file for the manifest, dataDir stores file contents.
func NewManifestStore(storePath, dataDir string) (*ManifestStore, error) {
	if err := os.MkdirAll(filepath.Dir(storePath), 0700); err != nil {
		return nil, fmt.Errorf("creating manifest store dir: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}
	ms := &ManifestStore{
		storePath: storePath,
		dataDir:   dataDir,
		nodeSync:  make(map[string]*types.SyncNodeStatus),
	}
	if err := ms.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return ms, nil
}

// Load reads the manifest from disk.
func (ms *ManifestStore) Load() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	data, err := os.ReadFile(ms.storePath)
	if err != nil {
		return err
	}
	var m types.SyncManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}
	ms.manifest = m
	return nil
}

// save persists the manifest to disk atomically. Caller must hold mu.
func (ms *ManifestStore) save() error {
	data, err := json.MarshalIndent(ms.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	tmp := fmt.Sprintf("%s.tmp.%d", ms.storePath, time.Now().UnixNano())
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("creating temp manifest: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("writing manifest: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("syncing manifest: %w", err)
	}
	f.Close()
	if err := os.Rename(tmp, ms.storePath); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming manifest: %w", err)
	}
	return nil
}

// GetManifest returns a deep copy of the current manifest.
func (ms *ManifestStore) GetManifest() types.SyncManifest {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	m := ms.manifest
	m.Files = make([]types.SyncFileEntry, len(ms.manifest.Files))
	copy(m.Files, ms.manifest.Files)
	return m
}

// GetFileContent reads a synced file from the data directory.
func (ms *ManifestStore) GetFileContent(path string) ([]byte, error) {
	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return nil, fmt.Errorf("invalid path: %s", path)
	}
	return os.ReadFile(filepath.Join(ms.dataDir, clean))
}

// ApplyPush processes a push request from a node.
// Files are written to dataDir, conflicts create .conflict-{nodeId} files.
func (ms *ManifestStore) ApplyPush(req types.SyncPushRequest) (*types.SyncPushResponse, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	resp := &types.SyncPushResponse{}
	fileMap := make(map[string]int) // path -> index in manifest.Files
	for i, f := range ms.manifest.Files {
		fileMap[f.Path] = i
	}

	for _, pf := range req.Files {
		clean := filepath.Clean(pf.Path)
		if strings.Contains(clean, "..") {
			continue
		}
		target := filepath.Join(ms.dataDir, clean)

		if pf.Delete {
			os.Remove(target)
			if idx, ok := fileMap[clean]; ok {
				ms.manifest.Files = append(ms.manifest.Files[:idx], ms.manifest.Files[idx+1:]...)
				// Rebuild map after removal.
				fileMap = make(map[string]int)
				for i, f := range ms.manifest.Files {
					fileMap[f.Path] = i
				}
			}
			resp.Accepted = append(resp.Accepted, clean)
			continue
		}

		// Check for conflict: file exists with different hash.
		if idx, ok := fileMap[clean]; ok {
			existing := ms.manifest.Files[idx]
			if existing.SHA256 != pf.SHA256 && existing.SHA256 != "" {
				// Conflict — keep both versions.
				conflictPath := clean + ".conflict-" + req.NodeID
				conflictTarget := filepath.Join(ms.dataDir, conflictPath)
				if err := os.MkdirAll(filepath.Dir(conflictTarget), 0700); err == nil {
					os.WriteFile(conflictTarget, []byte(pf.Content), 0644)
				}
				resp.Conflicts = append(resp.Conflicts, types.SyncConflict{
					Path:       clean,
					NodeID:     req.NodeID,
					Resolution: "kept_both",
				})
				continue
			}
		}

		// Write file.
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			continue
		}
		if err := os.WriteFile(target, []byte(pf.Content), 0644); err != nil {
			continue
		}

		hash := HashBytes([]byte(pf.Content))
		entry := types.SyncFileEntry{
			Path:    clean,
			SHA256:  hash,
			Size:    int64(len(pf.Content)),
			ModTime: time.Now(),
		}

		if idx, ok := fileMap[clean]; ok {
			ms.manifest.Files[idx] = entry
		} else {
			ms.manifest.Files = append(ms.manifest.Files, entry)
			fileMap[clean] = len(ms.manifest.Files) - 1
		}
		resp.Accepted = append(resp.Accepted, clean)
	}

	ms.manifest.Version++
	ms.manifest.UpdatedAt = time.Now()
	resp.Version = ms.manifest.Version

	if err := ms.save(); err != nil {
		return resp, fmt.Errorf("saving manifest: %w", err)
	}

	// Update node sync status.
	ms.nodeSync[req.NodeID] = &types.SyncNodeStatus{
		NodeID:          req.NodeID,
		LastSyncAt:      time.Now(),
		ManifestVersion: ms.manifest.Version,
		FileCount:       len(resp.Accepted),
		HasConflict:     len(resp.Conflicts) > 0,
	}

	return resp, nil
}

// SeedFromWorkspace initializes the manifest from a local workspace directory.
func (ms *ManifestStore) SeedFromWorkspace(wsDir string) error {
	entries, err := ScanWorkspaceDir(wsDir)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	for _, e := range entries {
		src := filepath.Join(wsDir, e.Path)
		dst := filepath.Join(ms.dataDir, e.Path)
		if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			continue
		}
	}

	ms.manifest.Files = entries
	ms.manifest.Version++
	ms.manifest.UpdatedAt = time.Now()
	return ms.save()
}

// GetNodeSyncStatuses returns sync status for all known nodes.
func (ms *ManifestStore) GetNodeSyncStatuses() []types.SyncNodeStatus {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	out := make([]types.SyncNodeStatus, 0, len(ms.nodeSync))
	for _, s := range ms.nodeSync {
		out = append(out, *s)
	}
	return out
}

// RecordNodeSync updates the sync status for a node (called after pull).
func (ms *ManifestStore) RecordNodeSync(nodeID string, version int64, fileCount int) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.nodeSync[nodeID] = &types.SyncNodeStatus{
		NodeID:          nodeID,
		LastSyncAt:      time.Now(),
		ManifestVersion: version,
		FileCount:       fileCount,
	}
}

// --- Pure functions ---

// ScanWorkspaceDir scans a workspace directory for syncable files.
func ScanWorkspaceDir(wsDir string) ([]types.SyncFileEntry, error) {
	var entries []types.SyncFileEntry

	// Fixed identity + memory files.
	for _, name := range syncablePatterns {
		p := filepath.Join(wsDir, name)
		hash, size, err := HashFile(p)
		if err != nil {
			continue
		}
		modTime := time.Now()
		if info, err := os.Stat(p); err == nil {
			modTime = info.ModTime()
		}
		entries = append(entries, types.SyncFileEntry{
			Path:    name,
			SHA256:  hash,
			Size:    size,
			ModTime: modTime,
		})
	}

	// memory/*.md
	memDir := filepath.Join(wsDir, "memory")
	if dirEntries, err := os.ReadDir(memDir); err == nil {
		for _, e := range dirEntries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			relPath := filepath.Join("memory", e.Name())
			p := filepath.Join(wsDir, relPath)
			hash, size, err := HashFile(p)
			if err != nil {
				continue
			}
			modTime := time.Now()
			if info, err := os.Stat(p); err == nil {
				modTime = info.ModTime()
			}
			entries = append(entries, types.SyncFileEntry{
				Path:    relPath,
				SHA256:  hash,
				Size:    size,
				ModTime: modTime,
			})
		}
	}

	// skills/**/SKILL.md and scripts
	skillsDir := filepath.Join(wsDir, "skills")
	if _, err := os.Stat(skillsDir); err == nil {
		filepath.Walk(skillsDir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(wsDir, p)
			hash, size, herr := HashFile(p)
			if herr != nil {
				return nil
			}
			entries = append(entries, types.SyncFileEntry{
				Path:    rel,
				SHA256:  hash,
				Size:    size,
				ModTime: info.ModTime(),
			})
			return nil
		})
	}

	return entries, nil
}

// HashFile computes the SHA-256 hash of a file.
func HashFile(path string) (hash string, size int64, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	return HashBytes(data), int64(len(data)), nil
}

// HashBytes computes the SHA-256 hash of a byte slice.
func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// DiffManifests compares a remote manifest against local file entries.
// Returns files that need to be downloaded and files that need to be uploaded.
func DiffManifests(remote types.SyncManifest, local []types.SyncFileEntry) (toDownload, toUpload []types.SyncFileEntry) {
	remoteMap := make(map[string]types.SyncFileEntry)
	for _, f := range remote.Files {
		remoteMap[f.Path] = f
	}
	localMap := make(map[string]types.SyncFileEntry)
	for _, f := range local {
		localMap[f.Path] = f
	}

	// Files in remote but not local, or different hash → download.
	for _, rf := range remote.Files {
		lf, ok := localMap[rf.Path]
		if !ok || lf.SHA256 != rf.SHA256 {
			toDownload = append(toDownload, rf)
		}
	}

	// Files in local but not remote, or different hash → upload.
	for _, lf := range local {
		rf, ok := remoteMap[lf.Path]
		if !ok || rf.SHA256 != lf.SHA256 {
			toUpload = append(toUpload, lf)
		}
	}

	return
}
