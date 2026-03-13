'use client'

import type { ChatMessage } from '@/lib/types'
import { formatTime } from '@/lib/utils'

interface MessageBubbleProps {
  message: ChatMessage
}

export function MessageBubble({ message }: MessageBubbleProps) {
  const isUser = message.source === 'user'

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
            <span className="text-cyan">[{message.node_id ?? 'node'}]</span>
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
