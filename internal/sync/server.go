package sync

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/SallyKAN/claw-mesh/internal/types"
)

// SyncServer handles sync-related HTTP endpoints on the coordinator.
type SyncServer struct {
	store *ManifestStore
}

// NewSyncServer creates a new sync server backed by the given store.
func NewSyncServer(store *ManifestStore) *SyncServer {
	return &SyncServer{store: store}
}

// HandleGetManifest serves GET /api/v1/sync/manifest.
func (s *SyncServer) HandleGetManifest(w http.ResponseWriter, r *http.Request) {
	m := s.store.GetManifest()
	writeJSON(w, http.StatusOK, m)
}

// HandleGetFile serves GET /api/v1/sync/file?path=xxx.
func (s *SyncServer) HandleGetFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path parameter required"})
		return
	}
	data, err := s.store.GetFileContent(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("file not found: %s", path)})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// HandlePush serves POST /api/v1/sync/push.
func (s *SyncServer) HandlePush(w http.ResponseWriter, r *http.Request) {
	var req types.SyncPushRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.NodeID == "" || len(req.Files) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id and files are required"})
		return
	}

	resp, err := s.store.ApplyPush(req)
	if err != nil {
		log.Printf("sync/push: error from node %s: %v", req.NodeID, err)
	}
	log.Printf("sync/push: node %s pushed %d files, %d accepted, %d conflicts",
		req.NodeID, len(req.Files), len(resp.Accepted), len(resp.Conflicts))
	writeJSON(w, http.StatusOK, resp)
}

// HandleStatus serves GET /api/v1/sync/status.
func (s *SyncServer) HandleStatus(w http.ResponseWriter, r *http.Request) {
	statuses := s.store.GetNodeSyncStatuses()
	if statuses == nil {
		statuses = []types.SyncNodeStatus{}
	}
	writeJSON(w, http.StatusOK, statuses)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
