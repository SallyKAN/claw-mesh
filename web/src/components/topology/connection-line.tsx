'use client'

interface ConnectionLineProps {
  x1: number
  y1: number
  x2: number
  y2: number
  active: boolean
}

export function ConnectionLine({ x1, y1, x2, y2, active }: ConnectionLineProps) {
  return (
    <line
      x1={x1}
      y1={y1}
      x2={x2}
      y2={y2}
      stroke={active ? 'var(--color-green)' : 'var(--color-border)'}
      strokeWidth={active ? 1.5 : 1}
      strokeDasharray="4 4"
      strokeOpacity={active ? 0.8 : 0.3}
    >
      {active && (
        <animate
          attributeName="stroke-dashoffset"
          from="8"
          to="0"
          dur="0.6s"
          repeatCount="indefinite"
        />
      )}
    </line>
  )
}
