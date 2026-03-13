'use client'

import { useCallback } from 'react'
import { useChatStore } from '@/stores/chat'
import { useNodesStore } from '@/stores/nodes'
import { usePolling } from '@/hooks/use-polling'
import { MessageList } from '@/components/chat/message-list'
import { ChatInput } from '@/components/chat/chat-input'

export default function ChatPage() {
  const messages = useChatStore((s) => s.messages)
  const sending = useChatStore((s) => s.sending)
  const routeTarget = useChatStore((s) => s.routeTarget)
  const send = useChatStore((s) => s.send)
  const setRouteTarget = useChatStore((s) => s.setRouteTarget)

  const nodes = useNodesStore((s) => s.nodes)
  const fetchNodes = useNodesStore((s) => s.fetch)

  usePolling(useCallback(() => fetchNodes(), [fetchNodes]))

  const nodeIds = nodes.map((n) => n.id)

  return (
    <div className="flex flex-col h-[calc(100vh-2.5rem)]">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border">
        <h1 className="text-xs font-bold uppercase tracking-widest text-green">
          MESSAGES
        </h1>
        <span className="text-xs font-mono text-text-dim">
          route: <span className={routeTarget === 'auto' ? 'text-amber' : 'text-cyan'}>{routeTarget}</span>
          {' // '}
          <span className="text-muted">{nodes.filter((n) => n.status === 'online').length} online</span>
        </span>
      </div>

      {/* Messages */}
      <MessageList messages={messages} sending={sending} />

      {/* Input */}
      <ChatInput
        onSend={send}
        routeTarget={routeTarget}
        onRouteChange={setRouteTarget}
        nodes={nodeIds}
        disabled={sending}
      />
    </div>
  )
}
