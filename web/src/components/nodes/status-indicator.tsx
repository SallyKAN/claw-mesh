'use client'

import { cn } from '@/lib/utils'

interface StatusIndicatorProps {
  status: 'online' | 'offline' | 'busy'
}

const statusConfig = {
  online: {
    dot: '\u25CF',
    label: 'ON',
    dotClass: 'text-green animate-[blink_2s_ease-in-out_infinite]',
    labelClass: 'text-green',
  },
  offline: {
    dot: '\u25CB',
    label: 'OFF',
    dotClass: 'text-muted',
    labelClass: 'text-muted',
  },
  busy: {
    dot: '\u25C9',
    label: 'BUSY',
    dotClass: 'text-amber',
    labelClass: 'text-amber',
  },
} as const

export function StatusIndicator({ status }: StatusIndicatorProps) {
  const config = statusConfig[status]

  return (
    <span className="inline-flex items-center gap-1.5" role="status" aria-label={`Node status: ${status}`}>
      <span className={cn('text-sm leading-none', config.dotClass)}>{config.dot}</span>
      <span className={cn('text-[10px] font-bold uppercase tracking-wider', config.labelClass)}>
        {config.label}
      </span>
    </span>
  )
}
