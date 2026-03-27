'use client'

import { Menu } from 'lucide-react'
import { useUIStore } from '@/stores/ui'
import { useNodesStore } from '@/stores/nodes'
import { useRulesStore } from '@/stores/rules'

export function Topbar() {
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)
  const nodes = useNodesStore((s) => s.nodes)
  const rules = useRulesStore((s) => s.rules)

  const onlineCount = nodes.filter((n) => n.status === 'online').length
  const totalCount = nodes.length

  return (
    <header className="fixed top-0 left-0 right-0 h-10 bg-surface border-b border-border z-30 flex items-center justify-between px-4">
      {/* Left: toggle */}
      <button
        onClick={toggleSidebar}
        className="text-text-dim hover:text-green transition-colors"
        aria-label="Toggle sidebar"
      >
        <Menu size={16} />
      </button>

      {/* Center: stats */}
      <div className="text-xs uppercase tracking-widest text-text-dim">
        <span>
          NODES: <span className="text-text">{onlineCount}/{totalCount}</span>
        </span>
        <span className="mx-3 text-border-bright">|</span>
        <span>
          ONLINE: <span className="text-green">{onlineCount}</span>
        </span>
        <span className="mx-3 text-border-bright">|</span>
        <span>
          RULES: <span className="text-text">{rules.length}</span>
        </span>
      </div>

      {/* Right: branding */}
      <span className="text-green text-xs font-bold uppercase tracking-widest">
        claw-mesh
      </span>
    </header>
  )
}
