'use client'

export function TypingIndicator() {
  return (
    <div className="flex items-center gap-1 py-1 font-mono text-sm text-green">
      <span className="text-text-dim">[mesh]</span>
      <span
        className="inline-block w-1.5 h-1.5 bg-green rounded-full"
        style={{ animation: 'blink 1s step-end infinite' }}
      />
      <span
        className="inline-block w-1.5 h-1.5 bg-green rounded-full"
        style={{ animation: 'blink 1s step-end infinite 0.2s' }}
      />
      <span
        className="inline-block w-1.5 h-1.5 bg-green rounded-full"
        style={{ animation: 'blink 1s step-end infinite 0.4s' }}
      />
    </div>
  )
}
