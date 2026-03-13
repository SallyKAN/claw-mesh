import type { Node, RoutingRule, NewRule, RouteResponse, SyncManifest, SyncNodeStatus } from './types'

declare global {
  interface Window {
    __TOKEN__?: string
  }
}

function getToken(): string {
  return window.__TOKEN__ ?? ''
}

async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken()
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  })
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText)
    throw new Error(`API ${res.status}: ${text}`)
  }
  return res.json() as Promise<T>
}

export const meshApi = {
  nodes: {
    list: () => api<Node[]>('/api/v1/nodes'),
    get: (id: string) => api<Node>(`/api/v1/nodes/${id}`),
    remove: (id: string) => api<void>(`/api/v1/nodes/${id}`, { method: 'DELETE' }),
  },
  route: {
    auto: (content: string) => api<RouteResponse>('/api/v1/route', {
      method: 'POST',
      body: JSON.stringify({ content }),
    }),
    toNode: (nodeId: string, content: string) => api<RouteResponse>(`/api/v1/route/${nodeId}`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    }),
  },
  rules: {
    list: () => api<RoutingRule[]>('/api/v1/rules'),
    add: (rule: NewRule) => api<RoutingRule>('/api/v1/rules', {
      method: 'POST',
      body: JSON.stringify(rule),
    }),
    remove: (id: string) => api<void>(`/api/v1/rules/${id}`, { method: 'DELETE' }),
  },
  sync: {
    manifest: () => api<SyncManifest>('/api/v1/sync/manifest'),
    status: () => api<SyncNodeStatus[]>('/api/v1/sync/status'),
  },
}
