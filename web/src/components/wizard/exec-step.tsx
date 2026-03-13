'use client'

import { useState } from 'react'

interface ExecStepConfig {
  networkType: 'lan' | 'public'
  coordinatorUrl: string
  name: string
  tags: string
}

interface ExecStepProps {
  config: ExecStepConfig
}

export function ExecStep({ config }: ExecStepProps) {
  const [copied, setCopied] = useState(false)
  const [sshHost, setSshHost] = useState('')
  const [sshUser, setSshUser] = useState('')

  const parts = ['claw-mesh join', config.coordinatorUrl || '<coordinator-url>']
  if (config.name) parts.push(`--name "${config.name}"`)
  if (config.tags) parts.push(`--tags "${config.tags}"`)
  const command = parts.join(' ')

  const handleCopy = async () => {
    await navigator.clipboard.writeText(command)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-6">
      <div className="text-[10px] font-bold uppercase tracking-widest text-text-dim mb-4">
        EXECUTE
      </div>

      {/* Command preview */}
      <div className="space-y-2">
        <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
          JOIN COMMAND
        </label>
        <div className="border border-border bg-bg p-4 font-mono text-sm text-green break-all">
          <span className="text-muted select-none">$ </span>
          {command}
        </div>
        <button
          type="button"
          onClick={handleCopy}
          className="border border-green text-green bg-transparent hover:bg-green hover:text-bg px-4 py-2 text-xs font-bold uppercase tracking-widest transition-colors"
        >
          {copied ? 'COPIED' : 'COPY'}
        </button>
      </div>

      {/* SSH deploy */}
      <div className="border-t border-border pt-6 space-y-4">
        <div className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
          DEPLOY VIA SSH (OPTIONAL)
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
              HOST
            </label>
            <input
              type="text"
              value={sshHost}
              onChange={(e) => setSshHost(e.target.value)}
              placeholder="192.168.1.100"
              className="flex h-9 w-full border border-border bg-surface px-3 py-1 text-sm font-mono text-text transition-colors placeholder:text-muted focus-visible:outline-none focus-visible:border-green"
            />
          </div>
          <div className="space-y-2">
            <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
              USER
            </label>
            <input
              type="text"
              value={sshUser}
              onChange={(e) => setSshUser(e.target.value)}
              placeholder="root"
              className="flex h-9 w-full border border-border bg-surface px-3 py-1 text-sm font-mono text-text transition-colors placeholder:text-muted focus-visible:outline-none focus-visible:border-green"
            />
          </div>
        </div>

        <button
          type="button"
          disabled={!sshHost || !sshUser}
          className="border border-amber text-amber bg-transparent hover:bg-amber hover:text-bg px-4 py-2 text-xs font-bold uppercase tracking-widest transition-colors disabled:opacity-50 disabled:pointer-events-none"
        >
          DEPLOY VIA SSH
        </button>
      </div>
    </div>
  )
}
