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
	Token  string `json:"token"`
}

// HeartbeatRequest is sent periodically by node agents.
type HeartbeatRequest struct {
	Status NodeStatus `json:"status"`
}
