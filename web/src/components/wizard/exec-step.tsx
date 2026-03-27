'use client'

import { useState, useRef, useEffect } from 'react'
import { cn } from '@/lib/utils'
import { meshApi } from '@/lib/api'
import type { ConfigValue } from './config-step'

interface ExecStepProps {
  config: {
    networkType: 'lan' | 'public'
    coordinatorUrl: string
  } & ConfigValue
}

type ExecTab = 'copy' | 'ssh'

export function ExecStep({ config }: ExecStepProps) {
  const [copied, setCopied] = useState(false)
  const [activeTab, setActiveTab] = useState<ExecTab>('copy')
  const [sshHost, setSshHost] = useState('')
  const [sshPort, setSshPort] = useState('22')
  const [sshUser, setSshUser] = useState('root')
  const [deploying, setDeploying] = useState(false)
  const [deployLog, setDeployLog] = useState<Array<{ type: 'info' | 'error'; text: string }>>([])
  const logRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    logRef.current?.scrollTo(0, logRef.current.scrollHeight)
  }, [deployLog])

  // Build join command (mirrors old buildJoinCmd)
  const token = typeof window !== 'undefined' ? window.__TOKEN__ : ''
  const parts = ['claw-mesh join', config.coordinatorUrl || '<coordinator-url>']
  if (token) parts.push(`--token ${token}`)
  if (config.name) parts.push(`--name "${config.name}"`)
  if (config.networkType === 'public' && config.endpoint) parts.push(`--endpoint "${config.endpoint}"`)
  if (config.tags) parts.push(`--tags "${config.tags}"`)
  if (config.autoInstall) parts.push('--auto-install')
  if (config.gatewayEndpoint) parts.push(`--gateway-endpoint "${config.gatewayEndpoint}"`)
  if (config.apiProvider) parts.push(`--api-provider ${config.apiProvider}`)
  if (config.apiKey) parts.push(`--api-key "${config.apiKey}"`)
  if (config.apiBase) parts.push(`--api-base "${config.apiBase}"`)
  if (config.apiModel) parts.push(`--api-model "${config.apiModel}"`)
  const command = parts.join(' ')

  const handleCopy = async () => {
    await navigator.clipboard.writeText(command)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleDeploy = async () => {
    if (!sshHost) return
    setDeploying(true)
    setDeployLog([{ type: 'info', text: 'Sending deploy request to AI agent...' }])

    const prompt = `Please help me add a new node to the claw-mesh cluster.

SSH into ${sshHost} as ${sshUser} (port ${sshPort}) and run:
${command}

If claw-mesh is not installed on the target machine, install it first with:
curl -fsSL https://get.claw-mesh.dev | sh

Make sure the join process starts successfully and the node registers with the coordinator.`

    try {
      const res = await meshApi.route.auto(prompt)
      setDeployLog((prev) => [
        ...prev,
        { type: 'info', text: 'AI response:' },
        { type: 'info', text: res.response || 'Deploy request sent.' },
      ])
    } catch (e) {
      setDeployLog((prev) => [
        ...prev,
        { type: 'error', text: `Failed: ${e instanceof Error ? e.message : String(e)}` },
      ])
    } finally {
      setDeploying(false)
    }
  }

  const inputClass = 'flex h-9 w-full border border-border bg-surface px-3 py-1 text-sm font-mono text-text transition-colors placeholder:text-muted focus-visible:outline-none focus-visible:border-green'

  return (
    <div className="space-y-6">
      <div className="text-[10px] font-bold uppercase tracking-widest text-text-dim mb-4">
        EXECUTE
      </div>

      {/* Tabs: Copy Command / SSH Deploy */}
      <div className="border border-border">
        <div className="flex border-b border-border">
          {(['copy', 'ssh'] as const).map((tab) => (
            <button
              key={tab}
              type="button"
              onClick={() => setActiveTab(tab)}
              className={cn(
                'flex-1 py-2.5 text-center text-[10px] font-bold uppercase tracking-widest transition-colors relative',
                activeTab === tab
                  ? 'text-cyan bg-cyan/5'
                  : 'text-muted hover:text-text-dim'
              )}
            >
              {tab === 'copy' ? 'COPY COMMAND' : 'SSH DEPLOY'}
              {activeTab === tab && (
                <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-cyan" />
              )}
            </button>
          ))}
        </div>

        {/* Copy Command panel */}
        {activeTab === 'copy' && (
          <div className="p-4 space-y-3">
            <div className="border border-border bg-bg p-4 font-mono text-sm text-green break-all whitespace-pre-wrap max-h-36 overflow-y-auto">
              {command}
            </div>
            <button
              type="button"
              onClick={handleCopy}
              className={cn(
                'w-full py-2.5 text-xs font-bold uppercase tracking-widest transition-colors border',
                copied
                  ? 'border-green text-green bg-green/5'
                  : 'border-cyan text-cyan hover:bg-cyan/10'
              )}
            >
              {copied ? 'COPIED!' : 'COPY COMMAND'}
            </button>
            <div className="text-[10px] text-muted text-center uppercase tracking-wider">
              RUN THIS ON THE TARGET MACHINE
            </div>
          </div>
        )}

        {/* SSH Deploy panel */}
        {activeTab === 'ssh' && (
          <div className="p-4 space-y-4">
            <div className="grid grid-cols-[1fr_100px] gap-3">
              <div className="space-y-2">
                <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
                  SSH HOST
                </label>
                <input
                  type="text"
                  value={sshHost}
                  onChange={(e) => setSshHost(e.target.value)}
                  placeholder="192.168.1.50"
                  className={inputClass}
                />
              </div>
              <div className="space-y-2">
                <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
                  PORT
                </label>
                <input
                  type="text"
                  value={sshPort}
                  onChange={(e) => setSshPort(e.target.value)}
                  placeholder="22"
                  className={inputClass}
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-[10px] font-bold uppercase tracking-widest text-text-dim">
                SSH USER
              </label>
              <input
                type="text"
                value={sshUser}
                onChange={(e) => setSshUser(e.target.value)}
                placeholder="root"
                className={inputClass}
              />
            </div>

            <button
              type="button"
              onClick={handleDeploy}
              disabled={!sshHost || deploying}
              className="w-full py-3 text-xs font-bold uppercase tracking-widest transition-all border border-amber text-amber hover:bg-amber hover:text-bg disabled:opacity-40 disabled:pointer-events-none"
            >
              {deploying ? 'DEPLOYING...' : 'DEPLOY VIA AI'}
            </button>

            {/* Deploy log */}
            {deployLog.length > 0 && (
              <div
                ref={logRef}
                className="border border-border bg-bg p-3 font-mono text-xs leading-relaxed max-h-40 overflow-y-auto whitespace-pre-wrap break-all"
              >
                {deployLog.map((entry, i) => (
                  <div key={i} className={entry.type === 'error' ? 'text-red' : 'text-cyan'}>
                    {entry.text}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Listening indicator */}
      <div className="flex items-center justify-center gap-3 py-2">
        <span
          className="w-2 h-2 rounded-full bg-cyan"
          style={{ animation: 'blink 1.5s ease-in-out infinite' }}
        />
        <span className="text-[10px] font-mono text-muted uppercase tracking-wider">
          Listening for incoming nodes...
        </span>
      </div>
    </div>
  )
}
