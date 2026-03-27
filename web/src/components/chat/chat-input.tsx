'use client'

import { useState } from 'react'
import { Send, ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ChatInputProps {
  onSend: (content: string) => void
  routeTarget: string
  onRouteChange: (target: string) => void
  nodes: string[]
  disabled: boolean
}

export function ChatInput({
  onSend,
  routeTarget,
  onRouteChange,
  nodes,
  disabled,
}: ChatInputProps) {
  const [value, setValue] = useState('')
  const [routeOpen, setRouteOpen] = useState(false)

  function handleSend() {
    const trimmed = value.trim()
    if (!trimmed || disabled) return
    onSend(trimmed)
    setValue('')
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div className="border-t border-border bg-surface px-4 py-3">
      <div className="flex items-center gap-2">
        {/* Route selector */}
        <div className="relative">
          <button
            type="button"
            onClick={() => setRouteOpen(!routeOpen)}
            className={cn(
              'flex items-center gap-1 h-9 px-3 border border-border bg-surface-2 text-xs font-mono uppercase tracking-wider transition-colors',
              'hover:border-border-bright focus:outline-none focus:border-green',
              routeTarget === 'auto' ? 'text-amber' : 'text-cyan'
            )}
            aria-label="Select route target"
            aria-expanded={routeOpen}
          >
            <span>{routeTarget === 'auto' ? 'AUTO' : routeTarget}</span>
            <ChevronDown size={12} className={cn('transition-transform', routeOpen && 'rotate-180')} />
          </button>

          {routeOpen && (
            <div className="absolute bottom-full left-0 mb-1 min-w-[140px] border border-border bg-surface-2 z-50">
              <button
                type="button"
                onClick={() => { onRouteChange('auto'); setRouteOpen(false) }}
                className={cn(
                  'block w-full text-left px-3 py-1.5 text-xs font-mono uppercase tracking-wider transition-colors',
                  'hover:bg-surface hover:text-green',
                  routeTarget === 'auto' ? 'text-amber' : 'text-text-dim'
                )}
              >
                AUTO
              </button>
              {nodes.map((id) => (
                <button
                  key={id}
                  type="button"
                  onClick={() => { onRouteChange(id); setRouteOpen(false) }}
                  className={cn(
                    'block w-full text-left px-3 py-1.5 text-xs font-mono uppercase tracking-wider transition-colors',
                    'hover:bg-surface hover:text-green',
                    routeTarget === id ? 'text-cyan' : 'text-text-dim'
                  )}
                >
                  {id}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Input */}
        <div className="flex-1 flex items-center gap-0 border border-border bg-bg focus-within:border-green transition-colors">
          <span className="pl-3 text-green font-mono text-sm select-none">&gt;</span>
          <input
            type="text"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={disabled}
            placeholder="type a message..."
            className="flex-1 h-9 bg-transparent px-2 text-sm font-mono text-text placeholder:text-muted focus:outline-none disabled:opacity-50"
            aria-label="Chat message input"
          />
        </div>

        {/* Send button */}
        <button
          type="button"
          onClick={handleSend}
          disabled={disabled || !value.trim()}
          className={cn(
            'h-9 px-4 border border-green text-green text-xs font-bold uppercase tracking-widest transition-colors',
            'hover:bg-green hover:text-bg',
            'disabled:opacity-40 disabled:pointer-events-none'
          )}
          aria-label="Send message"
        >
          <Send size={14} />
        </button>
      </div>
    </div>
  )
}
