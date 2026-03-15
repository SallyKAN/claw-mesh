'use client'

import Link from 'next/link'
import { Trash2, Send, Globe, Cpu, HardDrive, Tag, Wrench, Box, GitBranch } from 'lucide-react'
import type { Node } from '@/lib/types'
import { CapabilityBadges } from '@/components/nodes/capability-badges'

interface NodeDetailProps {
  node: Node
  onClose: () => void
  onRemove: (id: string) => void
}

function DetailRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-3 py-1.5">
      <span className="text-[10px] font-bold uppercase tracking-wider text-muted w-24 shrink-0 pt-0.5">
        {label}
      </span>
      <span className="text-xs text-text-dim">{children}</span>
    </div>
  )
}

export function NodeDetail({ node, onClose, onRemove }: NodeDetailProps) {
  const { capabilities } = node

  return (
    <div className="bg-surface border-x border-b border-border px-4 py-4">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-0">
        <div>
          <DetailRow label="OS / ARCH">
            {capabilities.os} / {capabilities.arch}
          </DetailRow>
          <DetailRow label="GPU">
            {capabilities.gpu ? (
              <span className="text-green">YES</span>
            ) : (
              <span className="text-muted">NO</span>
            )}
          </DetailRow>
          <DetailRow label="MEMORY">
            {capabilities.memory_gb} GB
          </DetailRow>
          <DetailRow label="ENDPOINT">
            <span className="text-cyan">{node.endpoint}</span>
          </DetailRow>
        </div>
        <div>
          <DetailRow label="TAGS">
            {capabilities.tags?.length ? capabilities.tags.join(', ') : '--'}
          </DetailRow>
          <DetailRow label="SKILLS">
            {capabilities.skills?.length ? capabilities.skills.join(', ') : '--'}
          </DetailRow>
          <DetailRow label="OPENCLAW">
            {node.openclaw_version || '--'}
          </DetailRow>
          <DetailRow label="SYNC VER">
            {node.sync_version ?? '--'}
          </DetailRow>
        </div>
      </div>

      <div className="mt-3 pt-3 border-t border-border">
        <CapabilityBadges capabilities={capabilities} />
      </div>

      <div className="mt-4 flex items-center gap-3">
        <button
          onClick={() => onRemove(node.id)}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider border border-red text-red hover:bg-red/10 transition-colors"
          aria-label={`Remove node ${node.name}`}
        >
          <Trash2 size={12} />
          REMOVE
        </button>
        <Link
          href="/chat"
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider border border-green text-green hover:bg-green/10 transition-colors"
        >
          <Send size={12} />
          SEND MSG
        </Link>
        <button
          onClick={onClose}
          className="ml-auto px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider border border-border text-text-dim hover:text-text hover:border-border-bright transition-colors"
        >
          CLOSE
        </button>
      </div>
    </div>
  )
}
