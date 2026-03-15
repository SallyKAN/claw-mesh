'use client'

import type { Node } from '@/lib/types'

interface CapabilityBadgesProps {
  capabilities: Node['capabilities']
}

const badgeBase =
  'inline-flex items-center rounded-none px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider border border-border bg-surface-2 text-text-dim'

export function CapabilityBadges({ capabilities }: CapabilityBadgesProps) {
  return (
    <div className="flex flex-wrap gap-1.5">
      <span className={badgeBase}>{capabilities.os}</span>
      <span className={badgeBase}>{capabilities.arch}</span>
      <span className={badgeBase}>MEM:{capabilities.memory_gb}GB</span>
      <span className={badgeBase}>
        GPU:{capabilities.gpu ? 'YES' : 'NO'}
      </span>
      {(capabilities.tags ?? []).map((tag) => (
        <span key={tag} className={badgeBase}>
          {tag}
        </span>
      ))}
    </div>
  )
}
