'use client'

interface CoordinatorHubProps {
  cx: number
  cy: number
}

export function CoordinatorHub({ cx, cy }: CoordinatorHubProps) {
  return (
    <g>
      {/* Pulse ring */}
      <circle
        cx={cx}
        cy={cy}
        r={32}
        fill="none"
        stroke="var(--color-green)"
        strokeWidth={1}
        opacity={0.3}
      >
        <animate
          attributeName="r"
          from="32"
          to="48"
          dur="2s"
          repeatCount="indefinite"
        />
        <animate
          attributeName="opacity"
          from="0.4"
          to="0"
          dur="2s"
          repeatCount="indefinite"
        />
      </circle>

      {/* Second pulse ring (offset) */}
      <circle
        cx={cx}
        cy={cy}
        r={32}
        fill="none"
        stroke="var(--color-green)"
        strokeWidth={1}
        opacity={0.2}
      >
        <animate
          attributeName="r"
          from="32"
          to="52"
          dur="2s"
          begin="1s"
          repeatCount="indefinite"
        />
        <animate
          attributeName="opacity"
          from="0.3"
          to="0"
          dur="2s"
          begin="1s"
          repeatCount="indefinite"
        />
      </circle>

      {/* Main circle */}
      <circle
        cx={cx}
        cy={cy}
        r={32}
        fill="var(--color-surface)"
        stroke="var(--color-green)"
        strokeWidth={2}
      />

      {/* Crab emoji */}
      <text
        x={cx}
        y={cy + 2}
        textAnchor="middle"
        dominantBaseline="middle"
        fontSize={22}
        aria-hidden="true"
      >
        🦀
      </text>

      {/* Label */}
      <text
        x={cx}
        y={cy + 50}
        textAnchor="middle"
        fill="var(--color-green)"
        fontSize={10}
        fontFamily="'IBM Plex Mono', monospace"
        fontWeight={700}
        letterSpacing="0.1em"
      >
        COORD
      </text>
    </g>
  )
}
