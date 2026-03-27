'use client'

import { useCallback } from 'react'
import { useSyncStore } from '@/stores/sync'
import { usePolling } from '@/hooks/use-polling'
import { SyncStatusCard } from '@/components/sync/sync-status-card'
import { ManifestViewer } from '@/components/sync/manifest-viewer'
import { Badge } from '@/components/ui/badge'

export default function SyncPage() {
  const { manifest, nodeStatuses, fetchManifest, fetchStatuses } =
    useSyncStore()

  const pollManifest = useCallback(() => fetchManifest(), [fetchManifest])
  const pollStatuses = useCallback(() => fetchStatuses(), [fetchStatuses])

  usePolling(pollManifest, 5000)
  usePolling(pollStatuses, 3000)

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-lg font-bold uppercase tracking-widest text-green">
          FILE SYNC
        </h1>
        {manifest && (
          <Badge variant="default">V{manifest.version}</Badge>
        )}
      </div>

      {/* Node sync status */}
      <section className="space-y-3">
        <h2 className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
          NODE SYNC STATUS
        </h2>
        <div className="border border-border">
          {nodeStatuses.length === 0 ? (
            <div className="px-4 py-6 text-center text-text-dim text-xs uppercase tracking-wider">
              NO NODES REPORTING
            </div>
          ) : (
            nodeStatuses.map((status) => (
              <SyncStatusCard key={status.node_id} status={status} />
            ))
          )}
        </div>
      </section>

      {/* Manifest files */}
      <section className="space-y-3">
        <h2 className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
          MANIFEST FILES
        </h2>
        {manifest ? (
          <ManifestViewer manifest={manifest} />
        ) : (
          <div className="border border-border px-4 py-6 text-center text-text-dim text-xs uppercase tracking-wider">
            LOADING MANIFEST...
          </div>
        )}
      </section>
    </div>
  )
}
