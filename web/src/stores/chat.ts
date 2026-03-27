import { create } from 'zustand'
import type { ChatMessage } from '@/lib/types'
import { useNodesStore } from './nodes'

interface ChatState {
  messages: ChatMessage[]
  sending: boolean
  routeTarget: string | 'auto'
  send: (content: string) => Promise<void>
  setRouteTarget: (target: string | 'auto') => void
  clear: () => void
}

function getToken(): string {
  return (typeof window !== 'undefined' && window.__TOKEN__) || ''
}

export const useChatStore = create<ChatState>((set, get) => ({
  messages: [],
  sending: false,
  routeTarget: 'auto',

  send: async (content: string) => {
    const userMsg: ChatMessage = {
      id: crypto.randomUUID(),
      content,
      source: 'user',
      timestamp: new Date().toISOString(),
    }
    set((state) => ({ messages: [...state.messages, userMsg], sending: true }))

    const placeholderId = crypto.randomUUID()
    const placeholder: ChatMessage = {
      id: placeholderId,
      content: '',
      source: 'node',
      timestamp: new Date().toISOString(),
    }
    set((state) => ({ messages: [...state.messages, placeholder] }))

    try {
      const { routeTarget } = get()
      const url = routeTarget === 'auto'
        ? '/api/v1/stream'
        : `/api/v1/stream/${routeTarget}`

      const token = getToken()
      const res = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ content, source: 'dashboard' }),
      })

      if (!res.ok) {
        const text = await res.text().catch(() => res.statusText)
        set((state) => ({
          messages: state.messages.map((m) =>
            m.id === placeholderId ? { ...m, content: `[Error] ${text}` } : m
          ),
        }))
        return
      }

      const reader = res.body?.getReader()
      if (!reader) return

      const decoder = new TextDecoder()
      let accumulated = ''
      let buf = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buf += decoder.decode(value, { stream: true })
        const lines = buf.split('\n')
        buf = lines.pop() ?? ''

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          const data = line.slice(6)
          if (data === '[DONE]') continue

          try {
            const chunk = JSON.parse(data)

            if (chunk.type === 'meta') {
              const nodeName = useNodesStore.getState().nodes.find((n) => n.id === chunk.node_id)?.name
              set((state) => ({
                messages: state.messages.map((m) =>
                  m.id === placeholderId
                    ? { ...m, node_id: chunk.node_id, node_name: nodeName }
                    : m
                ),
              }))
            }

            if (chunk.type === 'delta' && chunk.delta) {
              accumulated += chunk.delta
              const text = accumulated
              set((state) => ({
                messages: state.messages.map((m) =>
                  m.id === placeholderId ? { ...m, content: text } : m
                ),
              }))
            }

            if (chunk.type === 'error') {
              set((state) => ({
                messages: state.messages.map((m) =>
                  m.id === placeholderId
                    ? { ...m, content: `[Error] ${chunk.error}` }
                    : m
                ),
              }))
            }
          } catch {
            // Skip malformed JSON
          }
        }
      }
    } finally {
      set({ sending: false })
    }
  },

  setRouteTarget: (target: string | 'auto') => {
    set({ routeTarget: target })
  },

  clear: () => {
    set({ messages: [] })
  },
}))
