package types

import "time"

// NodeStatus represents the current state of a node.
type NodeStatus string

const (
	NodeStatusOnline  NodeStatus = "online"
	NodeStatusOffline NodeStatus = "offline"
	NodeStatusBusy    NodeStatus = "busy"
)

// Capabilities describes what a node can do.
type Capabilities struct {
	OS       string   `json:"os" yaml:"os"`
	Arch     string   `json:"arch" yaml:"arch"`
	GPU      bool     `json:"gpu" yaml:"gpu"`
	MemoryGB int      `json:"memory_gb" yaml:"memory_gb"`
	Tags     []string `json:"tags" yaml:"tags"`
	Skills   []string `json:"skills" yaml:"skills"`
}

// Node represents a single machine running an OpenClaw Gateway.
type Node struct {
	ID            string       `json:"id" yaml:"id"`
	Name          string       `json:"name" yaml:"name"`
	Endpoint      string       `json:"endpoint" yaml:"endpoint"`
	Capabilities  Capabilities `json:"capabilities" yaml:"capabilities"`
	Status        NodeStatus   `json:"status" yaml:"status"`
	LastHeartbeat time.Time    `json:"last_heartbeat" yaml:"last_heartbeat"`
}

// MatchCriteria defines what a routing rule matches against.
type MatchCriteria struct {
	RequiresGPU   *bool  `json:"requires_gpu,omitempty" yaml:"requires_gpu,omitempty"`
	RequiresOS    string `json:"requires_os,omitempty" yaml:"requires_os,omitempty"`
	RequiresSkill string `json:"requires_skill,omitempty" yaml:"requires_skill,omitempty"`
	Wildcard      *bool  `json:"wildcard,omitempty" yaml:"wildcard,omitempty"`
}

// RoutingRule defines how messages are routed to nodes.
type RoutingRule struct {
	ID       string        `json:"id" yaml:"id"`
	Match    MatchCriteria `json:"match" yaml:"match"`
	Target   string        `json:"target,omitempty" yaml:"target,omitempty"`
	Strategy string        `json:"strategy,omitempty" yaml:"strategy,omitempty"`
}

// Message represents a message flowing through the mesh.
type Message struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Source    string    `json:"source"`
	TargetNode string   `json:"target_node,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MessageResponse is the response returned after routing a message.
type MessageResponse struct {
	MessageID string `json:"message_id"`
	NodeID    string `json:"node_id"`
	Response  string `json:"response"`
}

// RegisterRequest is sent by a node agent to register with the coordinator.
type RegisterRequest struct {
	Name         string       `json:"name"`
	Endpoint     string       `json:"endpoint"`
	Capabilities Capabilities `json:"capabilities"`
}

// RegisterResponse is returned after successful registration.
type RegisterResponse struct {
	NodeID string `json:"node_id"`
	Token  string `json:"token,omitempty"`
}

// HeartbeatRequest is sent periodically by node agents.
type HeartbeatRequest struct {
	Status NodeStatus `json:"status"`
}

// ValidNodeStatus reports whether s is a known node status value.
func ValidNodeStatus(s NodeStatus) bool {
	switch s {
	case NodeStatusOnline, NodeStatusOffline, NodeStatusBusy:
		return true
	}
	return false
}

// --- OpenAI-compatible types for Gateway integration ---

// ChatMessage represents a single message in a chat completion request/response.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest is the OpenAI-compatible request sent to the Gateway.
type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

// ChatCompletionResponse is the OpenAI-compatible response from the Gateway.
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// --- Sync types (v0.2) ---

// SyncFileEntry is a single file record in a sync manifest.
type SyncFileEntry struct {
	Path    string    `json:"path"`
	SHA256  string    `json:"sha256"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// SyncManifest is the coordinator's authoritative file manifest.
type SyncManifest struct {
	Version   int64           `json:"version"`
	UpdatedAt time.Time       `json:"updated_at"`
	Files     []SyncFileEntry `json:"files"`
}

// SyncPushRequest is sent by a node to upload changed files.
type SyncPushRequest struct {
	NodeID string         `json:"node_id"`
	Files  []SyncPushFile `json:"files"`
}

// SyncPushFile is a single file in a push request.
type SyncPushFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	SHA256  string `json:"sha256"`
	Delete  bool   `json:"delete,omitempty"`
}

// SyncPushResponse is the result of a push operation.
type SyncPushResponse struct {
	Accepted  []string       `json:"accepted"`
	Conflicts []SyncConflict `json:"conflicts,omitempty"`
	Version   int64          `json:"version"`
}

// SyncConflict describes a file conflict during push.
type SyncConflict struct {
	Path       string `json:"path"`
	NodeID     string `json:"node_id"`
	Resolution string `json:"resolution"`
}

// SyncNodeStatus is a single node's sync state (for dashboard).
type SyncNodeStatus struct {
	NodeID          string    `json:"node_id"`
	LastSyncAt      time.Time `json:"last_sync_at"`
	ManifestVersion int64     `json:"manifest_version"`
	FileCount       int       `json:"file_count"`
	HasConflict     bool      `json:"has_conflict"`
}
