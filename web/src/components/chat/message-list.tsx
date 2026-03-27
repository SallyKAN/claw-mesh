'use client'

import { useRef, useEffect } from 'react'
import type { ChatMessage, Node } from '@/lib/types'
import { MessageBubble } from './message-bubble'
import { TypingIndicator } from './typing-indicator'

interface MessageListProps {
  messages: ChatMessage[]
  sending: boolean
  nodes?: Node[]
}

export function MessageList({ messages, sending, nodes = [] }: MessageListProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages.length, sending])

  return (
    <div className="flex-1 overflow-y-auto px-4 py-3">
      {messages.length === 0 && !sending && (
        <div className="text-text-dim font-mono text-sm py-8">
          <p>// no messages yet</p>
          <p>// type a command below to route to a node</p>
        </div>
      )}

      {messages.map((msg) => (
        <MessageBubble key={msg.id} message={msg} nodes={nodes} />
      ))}

      {sending && <TypingIndicator />}

      {!sending && (
        <span
          className="inline-block text-green font-mono text-sm"
          style={{ animation: 'blink 1s step-end infinite' }}
        >
          &#x2588;
        </span>
      )}

      <div ref={bottomRef} />
    </div>
  )
}
