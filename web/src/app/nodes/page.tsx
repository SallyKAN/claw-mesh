'use client'

import { useState } from 'react'
import Link from 'next/link'
import { Plus, Server } from 'lucide-react'
import { useNodesStore } from '@/stores/nodes'
import { usePolling } from '@/hooks/use-polling'
import { NodeCard } from '@/components/nodes/node-card'

export default function NodesPage() {
  const { nodes, loading, fetch, removeNode } = useNodesStore()
  const [expandedId, setExpandedId] = useState<string | null>(null)

  usePolling(fetch, 3000)

  const handleToggle = (id: string) => {
    setExpandedId((prev) => (prev === id ? null : id))
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <Server size={18} className="text-green" />
          <h1 className="text-lg font-bold uppercase tracking-widest text-text">
            NODES
          </h1>
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted border border-border px-2 py-0.5">
            {nodes.length}
          </span>
        </div>
        <Link
          href="/wizard"
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider border border-green text-green hover:bg-green/10 transition-colors"
        >
          <Plus size={12} />
          ADD
        </Link>
      </div>

      {/* Table */}
      <div className="border border-border">
        {/* Table Header */}
        <div className="grid grid-cols-[100px_1fr_120px_100px_100px] gap-4 px-4 py-2 border-b border-border bg-surface">
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted">ID</span>
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted">NAME</span>
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted">OS</span>
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted">STATUS</span>
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted text-right">LAST HB</span>
        </div>

        {/* Rows */}
        {nodes.length === 0 && !loading && (
          <div className="px-4 py-8 text-center text-xs text-muted uppercase tracking-wider">
            NO NODES REGISTERED
          </div>
        )}

        {loading && nodes.length === 0 && (
          <div className="px-4 py-8 text-center text-xs text-muted uppercase tracking-wider animate-pulse">
            LOADING...
          </div>
        )}

        {nodes.map((node) => (
          <NodeCard
            key={node.id}
            node={node}
            expanded={expandedId === node.id}
            onToggle={() => handleToggle(node.id)}
            onRemove={removeNode}
          />
        ))}
      </div>
    </div>
  )
}
