export interface Node {
  id: string
  name: string
  endpoint: string
  capabilities: {
    os: string
    arch: string
    gpu: boolean
    memory_gb: number
    tags: string[]
    skills: string[]
  }
  status: 'online' | 'offline' | 'busy'
  last_heartbeat: string
  openclaw_version: string
  sync_version: number
}

export interface RoutingRule {
  id: string
  match: Record<string, string>
  target: string
  strategy: string
}

export interface NewRule {
  match: Record<string, string>
  target: string
  strategy: string
}

export interface ChatMessage {
  id: string
  content: string
  source: 'user' | 'node'
  node_id?: string
  node_name?: string
  timestamp: string
}

export interface RouteResponse {
  node_id: string
  response: string
  routed_to: string
}

export interface TaskResponse {
  task_id: string
  node_id: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  response?: string
  partial_response?: string
  error?: string
}

export interface SyncFile {
  path: string
  sha256: string
  size: number
  modified: string
}

export interface SyncManifest {
  version: number
  files: SyncFile[]
}

export interface SyncNodeStatus {
  node_id: string
  node_name: string
  version: number
  file_count: number
  status: 'synced' | 'behind' | 'offline'
  last_sync: string
}

export interface MeshEvent {
  type: string
  timestamp: string
  data: unknown
}

export interface DashboardStats {
  total_nodes: number
  online_nodes: number
  synced_nodes: number
  rule_count: number
}
