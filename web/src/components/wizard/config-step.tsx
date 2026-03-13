'use client'

interface ConfigValue {
  name: string
  tags: string
}

interface ConfigStepProps {
  value: ConfigValue
  onChange: (value: ConfigValue) => void
}

export function ConfigStep({ value, onChange }: ConfigStepProps) {
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
          placeholder="my-node"
          className="flex h-9 w-full border border-border bg-surface px-3 py-1 text-sm font-mono text-text transition-colors placeholder:text-muted focus-visible:outline-none focus-visible:border-green"
        />
        <div className="text-[10px] text-muted uppercase tracking-wider">
          A FRIENDLY NAME FOR THIS NODE
        </div>
      </div>

      {/* Tags */}
      <div className="space-y-2">
        <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
          TAGS
        </label>
        <input
          type="text"
          value={value.tags}
          onChange={(e) => onChange({ ...value, tags: e.target.value })}
          placeholder="gpu, docker, linux"
          className="flex h-9 w-full border border-border bg-surface px-3 py-1 text-sm font-mono text-text transition-colors placeholder:text-muted focus-visible:outline-none focus-visible:border-green"
        />
        <div className="text-[10px] text-muted uppercase tracking-wider">
          COMMA-SEPARATED CAPABILITY TAGS
        </div>
      </div>
    </div>
  )
}
