package coordinator

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// excludedConfigKeys are openclaw.json top-level keys that are node-local
// and should NOT be distributed to new nodes via seed config.
var excludedConfigKeys = map[string]bool{
	"meta":     true,
	"wizard":   true,
	"channels": true,
	"gateway":  true,
	"bindings": true,
}

// seedWorkspaceFiles are the files from the workspace that should be
// synced to new nodes (identity layer + memory layer).
var seedWorkspaceFiles = []string{
	"SOUL.md",
	"IDENTITY.md",
	"AGENTS.md",
	"MEMORY.md",
}

// WorkspaceFile represents a single file in the seed workspace response.
type WorkspaceFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// WorkspaceSeedResponse is the response for GET /api/v1/seed/workspace.
type WorkspaceSeedResponse struct {
	Files []WorkspaceFile `json:"files"`
}

// handleSeedConfig serves GET /api/v1/seed/config.
// It reads the coordinator's local openclaw.json, strips node-local fields,
// and returns the filtered config for new nodes.
func (s *Server) handleSeedConfig(w http.ResponseWriter, r *http.Request) {
	cfgPath := s.resolveOpenClawConfigPath()
	if cfgPath == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "openclaw config not configured or not found",
		})
		return
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Printf("seed/config: failed to read %s: %v", cfgPath, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to read openclaw config: %v", err),
		})
		return
	}

	// Parse as generic JSON to filter keys.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("seed/config: failed to parse %s: %v", cfgPath, err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to parse openclaw config",
		})
		return
	}

	// Remove excluded keys.
	for k := range excludedConfigKeys {
		delete(raw, k)
	}

	log.Printf("seed/config: serving filtered config (%d keys) from %s", len(raw), cfgPath)
	writeJSON(w, http.StatusOK, raw)
}

// handleSeedWorkspace serves GET /api/v1/seed/workspace.
// It reads identity-layer and memory-layer files from the workspace
// and returns them as a JSON bundle.
func (s *Server) handleSeedWorkspace(w http.ResponseWriter, r *http.Request) {
	wsDir := s.resolveWorkspaceDir()
	if wsDir == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "workspace_dir not configured or not found",
		})
		return
	}

	var files []WorkspaceFile

	// Read fixed identity + memory files.
	for _, name := range seedWorkspaceFiles {
		p := filepath.Join(wsDir, name)
		content, err := os.ReadFile(p)
		if err != nil {
			continue // Skip missing files silently.
		}
		files = append(files, WorkspaceFile{Path: name, Content: string(content)})
	}

	// Read memory/*.md files.
	memDir := filepath.Join(wsDir, "memory")
	entries, err := os.ReadDir(memDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			p := filepath.Join(memDir, e.Name())
			content, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			files = append(files, WorkspaceFile{
				Path:    filepath.Join("memory", e.Name()),
				Content: string(content),
			})
		}
	}

	log.Printf("seed/workspace: serving %d files from %s", len(files), wsDir)
	writeJSON(w, http.StatusOK, WorkspaceSeedResponse{Files: files})
}

// resolveOpenClawConfigPath returns the path to openclaw.json.
func (s *Server) resolveOpenClawConfigPath() string {
	// Explicit config from coordinator settings.
	if p := s.cfg.OpenClawConfig; p != "" {
		expanded := expandHome(p)
		if _, err := os.Stat(expanded); err == nil {
			return expanded
		}
	}
	// Auto-detect common locations.
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".openclaw", "openclaw.json"),
		filepath.Join(home, ".config", "openclaw", "openclaw.json"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// resolveWorkspaceDir returns the workspace directory path.
func (s *Server) resolveWorkspaceDir() string {
	if p := s.cfg.WorkspaceDir; p != "" {
		expanded := expandHome(p)
		if info, err := os.Stat(expanded); err == nil && info.IsDir() {
			return expanded
		}
	}
	// Auto-detect: look for SOUL.md in common locations.
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "clawd"),
		filepath.Join(home, "openclaw"),
	}
	for _, c := range candidates {
		soul := filepath.Join(c, "SOUL.md")
		if _, err := os.Stat(soul); err == nil {
			return c
		}
	}
	return ""
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
