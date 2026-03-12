package sync

import (
	"log"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

// Watcher polls the workspace directory for file changes.
type Watcher struct {
	workspaceDir string
	interval     time.Duration
	onChange     func()
	stopCh       chan struct{}
	snapshot     map[string]string // path -> sha256
}

// NewWatcher creates a polling-based file watcher.
func NewWatcher(wsDir string, interval time.Duration, onChange func()) *Watcher {
	return &Watcher{
		workspaceDir: wsDir,
		interval:     interval,
		onChange:      onChange,
		stopCh:       make(chan struct{}),
	}
}

// Start begins the polling loop in a goroutine.
func (w *Watcher) Start() {
	// Take initial snapshot.
	w.snapshot = w.takeSnapshot()
	go w.loop()
}

// Stop terminates the polling loop.
func (w *Watcher) Stop() {
	select {
	case <-w.stopCh:
	default:
		close(w.stopCh)
	}
}

func (w *Watcher) loop() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			if w.hasChanges() {
				log.Printf("sync/watcher: changes detected in %s", w.workspaceDir)
				w.onChange()
			}
		}
	}
}

func (w *Watcher) hasChanges() bool {
	current := w.takeSnapshot()
	if len(current) != len(w.snapshot) {
		w.snapshot = current
		return true
	}
	for path, hash := range current {
		if w.snapshot[path] != hash {
			w.snapshot = current
			return true
		}
	}
	return false
}

func (w *Watcher) takeSnapshot() map[string]string {
	entries, err := ScanWorkspaceDir(w.workspaceDir)
	if err != nil {
		return w.snapshot
	}
	snap := make(map[string]string, len(entries))
	for _, e := range entries {
		snap[e.Path] = e.SHA256
	}
	return snap
}

// SnapshotFromEntries builds a snapshot map from scan entries (for testing).
func SnapshotFromEntries(entries []types.SyncFileEntry) map[string]string {
	snap := make(map[string]string, len(entries))
	for _, e := range entries {
		snap[e.Path] = e.SHA256
	}
	return snap
}
