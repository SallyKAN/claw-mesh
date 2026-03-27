'use client'

import type { Node } from '@/lib/types'
import { cn, timeAgo, osIcon } from '@/lib/utils'
import { StatusIndicator } from '@/components/nodes/status-indicator'
import { NodeDetail } from '@/components/nodes/node-detail'

interface NodeCardProps {
  node: Node
  expanded: boolean
  onToggle: () => void
  onRemove: (id: string) => void
}

export function NodeCard({ node, expanded, onToggle, onRemove }: NodeCardProps) {
  return (
    <div>
      <button
        onClick={onToggle}
        className={cn(
          'w-full grid grid-cols-[100px_1fr_120px_100px_100px] items-center gap-4 px-4 py-2.5 text-left border-b border-border transition-colors hover:bg-surface-2 focus:outline-none focus-visible:ring-1 focus-visible:ring-green',
          expanded && 'bg-surface-2'
        )}
        aria-expanded={expanded}
        aria-label={`Node ${node.name}, click to ${expanded ? 'collapse' : 'expand'} details`}
      >
        {/* ID */}
        <span className="text-xs text-muted font-mono truncate" title={node.id}>
          {node.id.slice(0, 8)}
        </span>

        {/* NAME */}
        <span className="text-xs text-text font-bold uppercase tracking-wider truncate">
          {node.name}
        </span>

        {/* OS */}
        <span className="text-xs text-text-dim flex items-center gap-1.5">
          <span className="text-sm leading-none">{osIcon(node.capabilities.os)}</span>
          <span className="uppercase">{node.capabilities.os}</span>
        </span>

        {/* STATUS */}
        <StatusIndicator status={node.status} />

        {/* LAST HB */}
        <span className="text-[10px] text-muted uppercase tracking-wider text-right">
          {timeAgo(node.last_heartbeat)}
        </span>
      </button>

      {expanded && (
        <NodeDetail
          node={node}
          onClose={onToggle}
          onRemove={onRemove}
        />
      )}
    </div>
  )
}
