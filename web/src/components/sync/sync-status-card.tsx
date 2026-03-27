'use client'

import type { SyncNodeStatus } from '@/lib/types'
import { timeAgo } from '@/lib/utils'

interface SyncStatusCardProps {
  status: SyncNodeStatus
}

function statusIcon(s: SyncNodeStatus['status']) {
  switch (s) {
    case 'synced':
      return <span className="text-green">&#10003;</span>
    case 'behind':
      return <span className="text-amber">&#9888;</span>
    case 'offline':
      return <span className="text-muted">&#9679;</span>
  }
}

export function SyncStatusCard({ status }: SyncStatusCardProps) {
  return (
    <div className="flex items-center justify-between border-b border-border px-4 py-3 font-mono text-sm hover:bg-surface-2 transition-colors">
      <div className="flex items-center gap-4">
        <span className="w-5 text-center">{statusIcon(status.status)}</span>
        <span className="text-text uppercase tracking-wider">
          {status.node_name || status.node_id}
        </span>
      </div>
      <div className="flex items-center gap-6 text-text-dim text-xs uppercase tracking-wider">
        <span>V{status.version}</span>
        <span>{status.file_count} FILES</span>
        <span
          className={
            status.status === 'synced'
              ? 'text-green'
              : status.status === 'behind'
                ? 'text-amber'
                : 'text-muted'
          }
        >
          {status.status.toUpperCase()}
        </span>
        <span className="text-muted">{timeAgo(status.last_sync)}</span>
      </div>
    </div>
  )
}
