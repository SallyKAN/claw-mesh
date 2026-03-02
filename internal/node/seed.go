package node

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// SeedWorkspaceResponse mirrors the coordinator's WorkspaceSeedResponse.
type SeedWorkspaceResponse struct {
	Files []SeedFile `json:"files"`
}

// SeedFile represents a single file in the seed workspace response.
type SeedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// FetchSeedConfig fetches the filtered openclaw.json from the coordinator
// and writes it to the local OpenClaw config path.
func FetchSeedConfig(coordinatorURL, token string) error {
	data, err := seedGet(coordinatorURL+"/api/v1/seed/config", token)
	if err != nil {
		return fmt.Errorf("fetch seed config: %w", err)
	}

	// Validate it's valid JSON.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("seed config is not valid JSON: %w", err)
	}

	// Determine local config path.
	cfgPath := localOpenClawConfigPath()
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	// If local config already exists, merge seed into it (seed fields win,
	// but preserve any local-only keys like channels/gateway).
	if existing, err := os.ReadFile(cfgPath); err == nil {
		var local map[string]json.RawMessage
		if json.Unmarshal(existing, &local) == nil {
			// Merge: seed overwrites shared keys, local keeps its own.
			for k, v := range raw {
				local[k] = v
			}
			merged, _ := json.MarshalIndent(local, "", "  ")
			if err := os.WriteFile(cfgPath, merged, 0600); err != nil {
				return fmt.Errorf("write merged config: %w", err)
			}
			log.Printf("seed/config: merged %d seed keys into %s", len(raw), cfgPath)
			return nil
		}
	}

	// No existing config — write seed as-is.
	pretty, _ := json.MarshalIndent(raw, "", "  ")
	if err := os.WriteFile(cfgPath, pretty, 0600); err != nil {
		return fmt.Errorf("write seed config: %w", err)
	}
	log.Printf("seed/config: wrote %d keys to %s", len(raw), cfgPath)
	return nil
}

// FetchSeedWorkspace fetches identity + memory layer files from the
// coordinator and writes them to the local workspace directory.
func FetchSeedWorkspace(coordinatorURL, token, workspaceDir string) error {
	data, err := seedGet(coordinatorURL+"/api/v1/seed/workspace", token)
	if err != nil {
		return fmt.Errorf("fetch seed workspace: %w", err)
	}

	var resp SeedWorkspaceResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parse seed workspace: %w", err)
	}

	if len(resp.Files) == 0 {
		log.Println("seed/workspace: no files to sync")
		return nil
	}

	for _, f := range resp.Files {
		target := filepath.Join(workspaceDir, f.Path)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			log.Printf("seed/workspace: failed to create dir for %s: %v", f.Path, err)
			continue
		}
		if err := os.WriteFile(target, []byte(f.Content), 0644); err != nil {
			log.Printf("seed/workspace: failed to write %s: %v", f.Path, err)
			continue
		}
	}

	log.Printf("seed/workspace: wrote %d files to %s", len(resp.Files), workspaceDir)
	return nil
}

// seedGet performs an authenticated GET request to the coordinator.
func seedGet(url, token string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// localOpenClawConfigPath returns the default local openclaw.json path.
func localOpenClawConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openclaw", "openclaw.json")
}
