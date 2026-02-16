package coordinator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

// Store provides persistent storage for routing rules.
// Data is stored as a JSON file on disk.
type Store struct {
	mu   sync.Mutex
	path string
}

// storeData is the on-disk JSON structure.
type storeData struct {
	Rules []*types.RoutingRule `json:"rules"`
}

// NewStore creates a store backed by the given file path.
// The parent directory is created if it doesn't exist.
func NewStore(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating store directory: %w", err)
	}
	return &Store{path: path}, nil
}

// LoadRules reads routing rules from disk.
// Returns an empty slice if the file doesn't exist.
func (s *Store) LoadRules() ([]*types.RoutingRule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading store: %w", err)
	}

	var sd storeData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("parsing store: %w", err)
	}
	return sd.Rules, nil
}

// SaveRules writes routing rules to disk atomically.
func (s *Store) SaveRules(rules []*types.RoutingRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sd := storeData{Rules: rules}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling store: %w", err)
	}

	// Atomic write: write to unique temp file, fsync, then rename.
	tmp := fmt.Sprintf("%s.tmp.%d", s.path, time.Now().UnixNano())
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("creating temp store: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("writing store: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("syncing store: %w", err)
	}
	f.Close()

	if err := os.Rename(tmp, s.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming store: %w", err)
	}
	return nil
}
