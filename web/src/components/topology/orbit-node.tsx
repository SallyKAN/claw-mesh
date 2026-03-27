'use client'

import { useState } from 'react'
import type { Node } from '@/lib/types'

interface OrbitNodeProps {
  node: Node
  cx: number
  cy: number
}

const statusColor: Record<Node['status'], string> = {
  online: 'var(--color-green)',
  offline: 'var(--color-muted)',
  busy: 'var(--color-amber)',
}

function osIcon(os: string): string {
  const lower = os.toLowerCase()
  if (lower.includes('darwin') || lower.includes('mac')) return '🍎'
  if (lower.includes('linux')) return '🐧'
  if (lower.includes('windows') || lower.includes('win')) return '🪟'
  return '💻'
}

export function OrbitNode({ node, cx, cy }: OrbitNodeProps) {
  const [hovered, setHovered] = useState(false)
  const color = statusColor[node.status]

  return (
    <g
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{ cursor: 'pointer' }}
    >
      {/* Node circle */}
      <circle
        cx={cx}
        cy={cy}
        r={22}
        fill="var(--color-surface-2)"
        stroke={hovered ? 'var(--color-green)' : 'var(--color-border-bright)'}
        strokeWidth={hovered ? 2 : 1.5}
      />

      {/* OS icon */}
      <text
        x={cx}
        y={cy + 2}
        textAnchor="middle"
        dominantBaseline="middle"
        fontSize={14}
        aria-hidden="true"
      >
        {osIcon(node.capabilities.os)}
      </text>

      {/* Status dot */}
      <circle
        cx={cx + 16}
        cy={cy - 16}
        r={4}
        fill={color}
        stroke="var(--color-bg)"
        strokeWidth={1.5}
      >
        {node.status === 'online' && (
          <animate
            attributeName="opacity"
            values="1;0.5;1"
            dur="2s"
            repeatCount="indefinite"
          />
        )}
      </circle>

      {/* Name label */}
      <text
        x={cx}
        y={cy + 36}
        textAnchor="middle"
        fill="var(--color-text-dim)"
        fontSize={9}
        fontFamily="'IBM Plex Mono', monospace"
        fontWeight={500}
        letterSpacing="0.05em"
      >
        {node.name.length > 12 ? node.name.slice(0, 11) + '…' : node.name}
      </text>

      {/* Tooltip on hover */}
      {hovered && (
        <g>
          <rect
            x={cx - 80}
            y={cy - 80}
            width={160}
            height={52}
            rx={2}
            fill="var(--color-surface)"
            stroke="var(--color-border-bright)"
            strokeWidth={1}
          />
          <text
            x={cx - 72}
            y={cy - 62}
            fill="var(--color-green)"
            fontSize={9}
            fontFamily="'IBM Plex Mono', monospace"
            fontWeight={700}
          >
            {node.name}
          </text>
          <text
            x={cx - 72}
            y={cy - 50}
            fill="var(--color-text-dim)"
            fontSize={8}
            fontFamily="'IBM Plex Mono', monospace"
          >
            {node.capabilities.os}/{node.capabilities.arch} | {node.capabilities.memory_gb}GB
          </text>
          <text
            x={cx - 72}
            y={cy - 39}
            fill={color}
            fontSize={8}
            fontFamily="'IBM Plex Mono', monospace"
            fontWeight={700}
            style={{ textTransform: 'uppercase' }}
          >
            {node.status}
            {node.capabilities.gpu ? ' | GPU' : ''}
          </text>
        </g>
      )}
    </g>
  )
}
