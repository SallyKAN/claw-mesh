'use client'

import { useState } from 'react'
import { ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface ConfigValue {
  name: string
  tags: string
  endpoint: string
  autoInstall: boolean
  gatewayEndpoint: string
  apiProvider: string
  apiKey: string
  apiBase: string
  apiModel: string
}

interface ConfigStepProps {
  value: ConfigValue
  onChange: (value: ConfigValue) => void
  networkType: 'lan' | 'public'
}

export function ConfigStep({ value, onChange, networkType }: ConfigStepProps) {
  const [advancedOpen, setAdvancedOpen] = useState(false)

  const inputClass = 'flex h-9 w-full border border-border bg-surface px-3 py-1 text-sm font-mono text-text transition-colors placeholder:text-muted focus-visible:outline-none focus-visible:border-green'

  return (
    <div className="space-y-6">
      <div className="text-[10px] font-bold uppercase tracking-widest text-text-dim mb-4">
        NODE CONFIGURATION
      </div>

      {/* Node name */}
      <div className="space-y-2">
        <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
          NODE NAME
        </label>
        <input
          type="text"
          value={value.name}
          onChange={(e) => onChange({ ...value, name: e.target.value })}
          placeholder="e.g. linux-gpu"
          className={inputClass}
        />
      </div>

      {/* Endpoint (only for public) */}
      {networkType === 'public' && (
        <div className="space-y-2">
          <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
            ENDPOINT (PUBLIC IP:PORT)
          </label>
          <input
            type="text"
            value={value.endpoint}
            onChange={(e) => onChange({ ...value, endpoint: e.target.value })}
            placeholder="e.g. 203.0.113.10:9121"
            className={inputClass}
          />
        </div>
      )}

      {/* Tags */}
      <div className="space-y-2">
        <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
          TAGS
        </label>
        <input
          type="text"
          value={value.tags}
          onChange={(e) => onChange({ ...value, tags: e.target.value })}
          placeholder="e.g. gpu, docker, coding"
          className={inputClass}
        />
        <div className="text-[10px] text-muted uppercase tracking-wider">
          COMMA-SEPARATED CAPABILITY TAGS
        </div>
      </div>

      {/* Auto-install toggle */}
      <div className="flex items-center justify-between py-1">
        <span className="text-xs font-mono text-text-dim">Auto-install runtime</span>
        <button
          type="button"
          onClick={() => onChange({ ...value, autoInstall: !value.autoInstall })}
          className={cn(
            'relative w-10 h-5 rounded-full transition-colors',
            value.autoInstall ? 'bg-green' : 'bg-border-bright'
          )}
        >
          <span
            className={cn(
              'absolute top-0.5 w-4 h-4 rounded-full transition-transform bg-white',
              value.autoInstall ? 'left-5' : 'left-0.5'
            )}
          />
        </button>
      </div>

      {/* Advanced section */}
      <div className="border-t border-border pt-4">
        <button
          type="button"
          onClick={() => setAdvancedOpen(!advancedOpen)}
          className="flex items-center gap-2 text-[10px] font-bold uppercase tracking-widest text-text-dim hover:text-text transition-colors"
        >
          <ChevronRight
            size={14}
            className={cn('transition-transform', advancedOpen && 'rotate-90')}
          />
          ADVANCED
        </button>

        {advancedOpen && (
          <div className="mt-4 space-y-4">
            <div className="space-y-2">
              <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
                GATEWAY ENDPOINT
              </label>
              <input
                type="text"
                value={value.gatewayEndpoint}
                onChange={(e) => onChange({ ...value, gatewayEndpoint: e.target.value })}
                placeholder="auto-discover"
                className={inputClass}
              />
            </div>

            <div className="space-y-2">
              <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
                API PROVIDER
              </label>
              <select
                value={value.apiProvider}
                onChange={(e) => onChange({ ...value, apiProvider: e.target.value })}
                className={cn(inputClass, 'cursor-pointer appearance-auto')}
              >
                <option value="">auto-detect</option>
                <option value="anthropic">Anthropic</option>
                <option value="openai">OpenAI</option>
                <option value="custom">Custom</option>
              </select>
            </div>

            <div className="space-y-2">
              <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
                API KEY
              </label>
              <input
                type="password"
                value={value.apiKey}
                onChange={(e) => onChange({ ...value, apiKey: e.target.value })}
                placeholder="sk-..."
                className={inputClass}
              />
            </div>

            <div className="space-y-2">
              <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
                API BASE URL
              </label>
              <input
                type="text"
                value={value.apiBase}
                onChange={(e) => onChange({ ...value, apiBase: e.target.value })}
                placeholder="https://api.example.com/v1"
                className={inputClass}
              />
            </div>

            <div className="space-y-2">
              <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
                API MODEL
              </label>
              <input
                type="text"
                value={value.apiModel}
                onChange={(e) => onChange({ ...value, apiModel: e.target.value })}
                placeholder="e.g. claude-sonnet-4-20250514"
                className={inputClass}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
