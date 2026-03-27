'use client'

import type { SyncManifest } from '@/lib/types'
import { formatBytes, timeAgo } from '@/lib/utils'

interface ManifestViewerProps {
  manifest: SyncManifest
}

export function ManifestViewer({ manifest }: ManifestViewerProps) {
  return (
    <div className="border border-border">
      {/* Table header */}
      <div className="grid grid-cols-[1fr_140px_80px_100px] gap-4 px-4 py-2 border-b border-border-bright bg-surface text-[10px] font-bold uppercase tracking-widest text-text-dim">
        <span>PATH</span>
        <span>SHA256</span>
        <span>SIZE</span>
        <span>MODIFIED</span>
      </div>

      {/* Table rows */}
      {manifest.files.length === 0 ? (
        <div className="px-4 py-6 text-center text-text-dim text-xs uppercase tracking-wider">
          NO FILES IN MANIFEST
        </div>
      ) : (
        manifest.files.map((file) => (
          <div
            key={file.path}
            className="grid grid-cols-[1fr_140px_80px_100px] gap-4 px-4 py-2 border-b border-border font-mono text-sm hover:bg-surface-2 transition-colors"
          >
            <span className="text-text truncate">{file.path}</span>
            <span className="text-text-dim">
              {file.sha256.slice(0, 8)}..
            </span>
            <span className="text-text-dim">{formatBytes(file.size)}</span>
            <span className="text-muted">{timeAgo(file.modified)}</span>
          </div>
        ))
      )}
    </div>
  )
}
