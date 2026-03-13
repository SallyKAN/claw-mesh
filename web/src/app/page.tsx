'use client'

import { useRef, useEffect } from 'react'
import { Server, Wifi, GitBranch, Activity } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useNodesStore } from '@/stores/nodes'
import { usePolling } from '@/hooks/use-polling'
import { TopologyCanvas } from '@/components/topology/topology-canvas'

interface StatCardProps {
  label: string
  value: number | string
  icon: React.ElementType
  color?: string
}

function StatCard({ label, value, icon: Icon, color = 'text-green' }: StatCardProps) {
  return (
    <div className="border border-border bg-surface px-4 py-3 flex items-center gap-3">
      <Icon size={16} className={cn('shrink-0', color)} />
      <div>
        <div className={cn('text-lg font-bold tabular-nums', color)}>{value}</div>
        <div className="text-[10px] uppercase tracking-widest text-text-dim">{label}</div>
      </div>
    </div>
  )
}

const MOCK_ACTIVITY = [
  { ts: '12:04:31', msg: 'node mac-mini heartbeat OK', type: 'info' as const },
  { ts: '12:04:28', msg: 'route /v1/chat → linux-gpu (capability: gpu)', type: 'route' as const },
  { ts: '12:04:22', msg: 'node linux-gpu status: online', type: 'info' as const },
  { ts: '12:04:15', msg: 'sync MEMORY.md → 3 nodes (v12)', type: 'sync' as const },
  { ts: '12:04:10', msg: 'route /v1/chat → mac-mini (capability: macos)', type: 'route' as const },
  { ts: '12:04:02', msg: 'node vps-east registered (linux/amd64)', type: 'info' as const },
  { ts: '12:03:58', msg: 'health check: 3/3 nodes responding', type: 'info' as const },
  { ts: '12:03:51', msg: 'route /v1/chat → linux-gpu (strategy: least-busy)', type: 'route' as const },
  { ts: '12:03:44', msg: 'node mac-mini heartbeat OK', type: 'info' as const },
  { ts: '12:03:30', msg: 'coordinator started on :9180', type: 'info' as const },
]

const typeColor: Record<string, string> = {
  info: 'text-text-dim',
  route: 'text-cyan',
  sync: 'text-amber',
}

export default function OverviewPage() {
  const nodes = useNodesStore((s) => s.nodes)
  const fetch = useNodesStore((s) => s.fetch)
  const logRef = useRef<HTMLDivElement>(null)

  usePolling(fetch, 3000)

  useEffect(() => {
    if (logRef.current) {
      logRef.current.scrollTop = 0
    }
  }, [])

  const onlineCount = nodes.filter((n) => n.status === 'online').length
  const busyCount = nodes.filter((n) => n.status === 'busy').length

  return (
    <div className="space-y-4">
      {/* Stats bar */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <StatCard label="Nodes" value={nodes.length} icon={Server} />
        <StatCard label="Online" value={onlineCount} icon={Wifi} color="text-green" />
        <StatCard label="Busy" value={busyCount} icon={Activity} color="text-amber" />
        <StatCard label="Routes" value={nodes.length > 0 ? 'auto' : '--'} icon={GitBranch} color="text-cyan" />
      </div>

      {/* Topology */}
      <div className="border border-border bg-surface">
        <div className="flex items-center justify-between px-4 py-2 border-b border-border">
          <span className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
            Mesh Topology
          </span>
          <span className="text-[10px] text-muted tabular-nums">
            {nodes.length} node{nodes.length !== 1 ? 's' : ''} registered
          </span>
        </div>
        <div className="p-2">
          <TopologyCanvas nodes={nodes} />
        </div>
      </div>

      {/* Activity log */}
      <div className="border border-border bg-surface">
        <div className="flex items-center justify-between px-4 py-2 border-b border-border">
          <span className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
            Activity Log
          </span>
          <span className="text-[10px] text-muted">
            live
            <span className="inline-block w-1.5 h-1.5 rounded-full bg-green ml-1.5 align-middle" style={{ animation: 'blink 1.5s step-end infinite' }} />
          </span>
        </div>
        <div
          ref={logRef}
          className="h-48 overflow-y-auto p-3 font-mono text-xs leading-relaxed"
          role="log"
          aria-label="Activity log"
        >
          {MOCK_ACTIVITY.map((entry, i) => (
            <div key={i} className="flex gap-2">
              <span className="text-green shrink-0">{entry.ts}</span>
              <span className="text-muted shrink-0">|</span>
              <span className={cn(typeColor[entry.type] ?? 'text-text-dim')}>
                {entry.msg}
              </span>
            </div>
          ))}
          <div className="text-muted mt-1">
            <span className="inline-block" style={{ animation: 'blink 1s step-end infinite' }}>_</span>
          </div>
        </div>
      </div>
    </div>
  )
}
