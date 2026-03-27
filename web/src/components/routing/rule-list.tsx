'use client'

import { X } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { RoutingRule } from '@/lib/types'

interface RuleListProps {
  rules: RoutingRule[]
  onRemove: (id: string) => void
}

export function RuleList({ rules, onRemove }: RuleListProps) {
  if (rules.length === 0) {
    return (
      <div className="border border-border px-4 py-6 text-center text-text-dim text-xs uppercase tracking-widest">
        NO ROUTING RULES CONFIGURED
      </div>
    )
  }

  return (
    <div className="border border-border">
      {/* Header */}
      <div className="flex items-center gap-4 px-4 py-2 border-b border-border bg-surface text-text-dim text-xs uppercase tracking-widest">
        <span className="w-8 shrink-0">#</span>
        <span className="flex-1">RULE</span>
        <span className="w-28 text-right">STRATEGY</span>
        <span className="w-8 shrink-0" />
      </div>

      {/* Rows */}
      {rules.map((rule, index) => {
        const matchEntries = Object.entries(rule.match)
        const matchStr = matchEntries.length > 0
          ? matchEntries.map(([k, v]) => `${k}=${v}`).join(', ')
          : '*'

        return (
          <div
            key={rule.id}
            className={cn(
              'flex items-center gap-4 px-4 py-2.5 text-sm group',
              index < rules.length - 1 && 'border-b border-border'
            )}
          >
            <span className="w-8 shrink-0 text-text-dim text-xs">
              {String(index + 1).padStart(2, '0')}
            </span>

            <span className="flex-1 truncate">
              <span className="text-text-dim">IF </span>
              <span className="text-amber">{matchStr}</span>
              <span className="text-text-dim"> → </span>
              <span className="text-green">{rule.target}</span>
            </span>

            <span className="w-28 text-right text-xs text-cyan uppercase tracking-wider">
              [{rule.strategy}]
            </span>

            <button
              onClick={() => onRemove(rule.id)}
              className="w-8 shrink-0 flex items-center justify-center text-text-dim hover:text-red transition-colors"
              aria-label={`Remove rule ${index + 1}`}
            >
              <X size={14} className="opacity-0 group-hover:opacity-100 transition-opacity" />
            </button>
          </div>
        )
      })}
    </div>
  )
}
