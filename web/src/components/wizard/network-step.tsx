'use client'

import { useEffect } from 'react'
import { cn } from '@/lib/utils'

interface NetworkValue {
  type: 'lan' | 'public'
  coordinatorUrl: string
}

interface NetworkStepProps {
  value: NetworkValue
  onChange: (value: NetworkValue) => void
}

export function NetworkStep({ value, onChange }: NetworkStepProps) {
  useEffect(() => {
    if (value.type === 'lan' && !value.coordinatorUrl) {
      onChange({ ...value, coordinatorUrl: window.location.origin })
    }
  }, [value, onChange])

  return (
    <div className="space-y-6">
      <div className="text-[10px] font-bold uppercase tracking-widest text-text-dim mb-4">
        NETWORK TYPE
      </div>

      <div className="grid grid-cols-2 gap-4">
        {/* LAN card */}
        <button
          type="button"
          onClick={() =>
            onChange({ type: 'lan', coordinatorUrl: window.location.origin })
          }
          className={cn(
            'border p-4 text-left transition-colors',
            value.type === 'lan'
              ? 'border-green bg-green/5'
              : 'border-border hover:border-border-bright'
          )}
        >
          <div
            className={cn(
              'text-xs font-bold uppercase tracking-widest mb-2',
              value.type === 'lan' ? 'text-green' : 'text-text'
            )}
          >
            LAN
          </div>
          <div className="text-[10px] text-text-dim uppercase tracking-wider">
            SAME LOCAL NETWORK
          </div>
        </button>

        {/* PUBLIC card */}
        <button
          type="button"
          onClick={() => onChange({ type: 'public', coordinatorUrl: '' })}
          className={cn(
            'border p-4 text-left transition-colors',
            value.type === 'public'
              ? 'border-green bg-green/5'
              : 'border-border hover:border-border-bright'
          )}
        >
          <div
            className={cn(
              'text-xs font-bold uppercase tracking-widest mb-2',
              value.type === 'public' ? 'text-green' : 'text-text'
            )}
          >
            PUBLIC
          </div>
          <div className="text-[10px] text-text-dim uppercase tracking-wider">
            PUBLIC IP / TUNNEL
          </div>
        </button>
      </div>

      {/* Coordinator URL */}
      <div className="space-y-2">
        <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
          COORDINATOR URL
        </label>
        <input
          type="text"
          value={value.coordinatorUrl}
          onChange={(e) =>
            onChange({ ...value, coordinatorUrl: e.target.value })
          }
          placeholder="https://coordinator.example.com"
          className="flex h-9 w-full border border-border bg-surface px-3 py-1 text-sm font-mono text-text transition-colors placeholder:text-muted focus-visible:outline-none focus-visible:border-green"
        />
      </div>
    </div>
  )
}
