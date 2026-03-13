'use client'

import type { Node } from '@/lib/types'
import { CoordinatorHub } from './coordinator-hub'
import { OrbitNode } from './orbit-node'
import { ConnectionLine } from './connection-line'

interface TopologyCanvasProps {
  nodes: Node[]
}

const CENTER_X = 300
const CENTER_Y = 200
const ORBIT_RADIUS = 130

function getOrbitPosition(index: number, total: number) {
  // Start from top (-PI/2) and distribute evenly
  const angle = -Math.PI / 2 + (2 * Math.PI * index) / Math.max(total, 1)
  return {
    x: CENTER_X + ORBIT_RADIUS * Math.cos(angle),
    y: CENTER_Y + ORBIT_RADIUS * Math.sin(angle),
  }
}

export function TopologyCanvas({ nodes }: TopologyCanvasProps) {
  return (
    <svg
      viewBox="0 0 600 400"
      className="w-full h-auto"
      role="img"
      aria-label="Mesh topology diagram"
    >
      {/* Background grid dots */}
      <defs>
        <pattern id="grid-dots" width="20" height="20" patternUnits="userSpaceOnUse">
          <circle cx="10" cy="10" r="0.5" fill="var(--color-border)" />
        </pattern>
      </defs>
      <rect width="600" height="400" fill="url(#grid-dots)" />

      {/* Orbit ring */}
      <circle
        cx={CENTER_X}
        cy={CENTER_Y}
        r={ORBIT_RADIUS}
        fill="none"
        stroke="var(--color-border)"
        strokeWidth={1}
        strokeDasharray="2 6"
        opacity={0.4}
      />

      {/* Connection lines (render behind nodes) */}
      {nodes.map((node, i) => {
        const pos = getOrbitPosition(i, nodes.length)
        return (
          <ConnectionLine
            key={`line-${node.id}`}
            x1={CENTER_X}
            y1={CENTER_Y}
            x2={pos.x}
            y2={pos.y}
            active={node.status === 'online' || node.status === 'busy'}
          />
        )
      })}

      {/* Coordinator hub */}
      <CoordinatorHub cx={CENTER_X} cy={CENTER_Y} />

      {/* Orbit nodes */}
      {nodes.map((node, i) => {
        const pos = getOrbitPosition(i, nodes.length)
        return (
          <OrbitNode
            key={node.id}
            node={node}
            cx={pos.x}
            cy={pos.y}
          />
        )
      })}

      {/* Empty state */}
      {nodes.length === 0 && (
        <text
          x={CENTER_X}
          y={CENTER_Y + 80}
          textAnchor="middle"
          fill="var(--color-text-dim)"
          fontSize={10}
          fontFamily="'IBM Plex Mono', monospace"
        >
          no nodes registered
        </text>
      )}
    </svg>
  )
}
