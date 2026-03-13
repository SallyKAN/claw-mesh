'use client'

import type { ChatMessage, Node } from '@/lib/types'
import { formatTime } from '@/lib/utils'

interface MessageBubbleProps {
  message: ChatMessage
  nodes?: Node[]
}

export function MessageBubble({ message, nodes = [] }: MessageBubbleProps) {
  const isUser = message.source === 'user'

  const nodeName = message.node_name
    ?? nodes.find((n) => n.id === message.node_id)?.name
    ?? message.node_id
    ?? 'node'

  return (
    <div className="flex items-start justify-between gap-4 py-0.5 font-mono text-sm leading-relaxed">
      <div className="flex-1 min-w-0">
        {isUser ? (
          <span>
            <span className="text-green font-bold">&gt; </span>
            <span className="text-green">{message.content}</span>
          </span>
        ) : (
          <span>
            <span className="text-cyan">[{nodeName}]</span>
            <span className="text-text"> {message.content}</span>
          </span>
        )}
      </div>
      <span className="shrink-0 text-xs text-text-dim tabular-nums">
        {formatTime(message.timestamp)}
      </span>
    </div>
  )
}
