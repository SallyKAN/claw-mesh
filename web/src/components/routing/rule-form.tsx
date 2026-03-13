'use client'

import { useState } from 'react'
import { Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { NewRule } from '@/lib/types'

interface RuleFormProps {
  onAdd: (rule: NewRule) => void
  nodes: string[]
}

export function RuleForm({ onAdd, nodes }: RuleFormProps) {
  const [matchKey, setMatchKey] = useState('')
  const [matchValue, setMatchValue] = useState('')
  const [target, setTarget] = useState('any')
  const [strategy, setStrategy] = useState('least-busy')

  const inputClass = cn(
    'w-full bg-surface border border-border px-3 py-2 text-sm text-text',
    'placeholder:text-muted uppercase tracking-wider',
    'focus:outline-none focus:border-green transition-colors'
  )

  const selectClass = cn(
    'w-full bg-surface border border-border px-3 py-2 text-sm text-text',
    'uppercase tracking-wider appearance-none cursor-pointer',
    'focus:outline-none focus:border-green transition-colors'
  )

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const match: Record<string, string> = {}
    if (matchKey.trim()) {
      match[matchKey.trim()] = matchValue.trim()
    }

    onAdd({
      match,
      target: target === 'any' ? '*' : target,
      strategy,
    })

    setMatchKey('')
    setMatchValue('')
    setTarget('any')
    setStrategy('least-busy')
  }

  return (
    <form onSubmit={handleSubmit} className="border border-border p-4 bg-surface">
      <div className="text-xs text-text-dim uppercase tracking-widest mb-4">
        NEW RULE
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mb-4">
        {/* Match key */}
        <div>
          <label htmlFor="rule-match-key" className="block text-xs text-text-dim uppercase tracking-widest mb-1.5">
            MATCH KEY
          </label>
          <input
            id="rule-match-key"
            type="text"
            value={matchKey}
            onChange={(e) => setMatchKey(e.target.value)}
            placeholder="requires_gpu"
            className={inputClass}
          />
        </div>

        {/* Match value */}
        <div>
          <label htmlFor="rule-match-value" className="block text-xs text-text-dim uppercase tracking-widest mb-1.5">
            MATCH VALUE
          </label>
          <input
            id="rule-match-value"
            type="text"
            value={matchValue}
            onChange={(e) => setMatchValue(e.target.value)}
            placeholder="true"
            className={inputClass}
          />
        </div>

        {/* Target */}
        <div>
          <label htmlFor="rule-target" className="block text-xs text-text-dim uppercase tracking-widest mb-1.5">
            TARGET
          </label>
          <select
            id="rule-target"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            className={selectClass}
          >
            <option value="any">ANY</option>
            {nodes.map((name) => (
              <option key={name} value={name}>
                {name.toUpperCase()}
              </option>
            ))}
          </select>
        </div>

        {/* Strategy */}
        <div>
          <label htmlFor="rule-strategy" className="block text-xs text-text-dim uppercase tracking-widest mb-1.5">
            STRATEGY
          </label>
          <select
            id="rule-strategy"
            value={strategy}
            onChange={(e) => setStrategy(e.target.value)}
            className={selectClass}
          >
            <option value="least-busy">LEAST-BUSY</option>
            <option value="round-robin">ROUND-ROBIN</option>
            <option value="random">RANDOM</option>
          </select>
        </div>
      </div>

      <button
        type="submit"
        className={cn(
          'flex items-center gap-2 border border-green px-4 py-2',
          'text-green text-xs uppercase tracking-widest font-bold',
          'hover:bg-green hover:text-bg transition-colors'
        )}
      >
        <Plus size={14} />
        SAVE RULE
      </button>
    </form>
  )
}
