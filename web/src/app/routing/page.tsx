'use client'

import { useState, useCallback } from 'react'
import { Plus, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useRulesStore } from '@/stores/rules'
import { useNodesStore } from '@/stores/nodes'
import { usePolling } from '@/hooks/use-polling'
import { RuleList } from '@/components/routing/rule-list'
import { RuleForm } from '@/components/routing/rule-form'
import type { NewRule } from '@/lib/types'

export default function RoutingPage() {
  const [showForm, setShowForm] = useState(false)

  const rules = useRulesStore((s) => s.rules)
  const addRule = useRulesStore((s) => s.add)
  const removeRule = useRulesStore((s) => s.remove)
  const fetchRules = useRulesStore((s) => s.fetch)

  const nodes = useNodesStore((s) => s.nodes)
  const fetchNodes = useNodesStore((s) => s.fetch)

  usePolling(useCallback(async () => { await fetchRules() }, [fetchRules]))
  usePolling(useCallback(async () => { await fetchNodes() }, [fetchNodes]))

  const nodeNames = nodes.map((n) => n.name)

  const handleAdd = async (rule: NewRule) => {
    await addRule(rule)
    setShowForm(false)
  }

  const handleRemove = async (id: string) => {
    await removeRule(id)
  }

  return (
    <div className="max-w-4xl">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-lg font-bold text-text uppercase tracking-widest">
          ROUTING RULES
        </h1>

        <button
          onClick={() => setShowForm(!showForm)}
          className={cn(
            'flex items-center gap-2 border px-3 py-1.5',
            'text-xs uppercase tracking-widest font-bold transition-colors',
            showForm
              ? 'border-red text-red hover:bg-red hover:text-bg'
              : 'border-green text-green hover:bg-green hover:text-bg'
          )}
        >
          {showForm ? <X size={14} /> : <Plus size={14} />}
          {showForm ? 'CANCEL' : 'ADD'}
        </button>
      </div>

      {/* Rule count */}
      <div className="text-xs text-text-dim uppercase tracking-widest mb-4">
        {rules.length} {rules.length === 1 ? 'RULE' : 'RULES'} ACTIVE
      </div>

      {/* Rule list */}
      <RuleList rules={rules} onRemove={handleRemove} />

      {/* Add form */}
      {showForm && (
        <div className="mt-4">
          <RuleForm onAdd={handleAdd} nodes={nodeNames} />
        </div>
      )}
    </div>
  )
}
