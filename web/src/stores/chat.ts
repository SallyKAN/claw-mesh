import { create } from 'zustand'
import type { ChatMessage } from '@/lib/types'
import { meshApi } from '@/lib/api'
import { useNodesStore } from './nodes'

interface ChatState {
  messages: ChatMessage[]
  sending: boolean
  routeTarget: string | 'auto'
  send: (content: string) => Promise<void>
  setRouteTarget: (target: string | 'auto') => void
  clear: () => void
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

    try {
      const { routeTarget } = get()
      const res = routeTarget === 'auto'
        ? await meshApi.route.auto(content)
        : await meshApi.route.toNode(routeTarget, content)

      const nodeMsg: ChatMessage = {
        id: crypto.randomUUID(),
        content: res.response,
        source: 'node',
        node_id: res.node_id,
        node_name: useNodesStore.getState().nodes.find((n) => n.id === res.node_id)?.name,
        timestamp: new Date().toISOString(),
      }
      set((state) => ({ messages: [...state.messages, nodeMsg] }))
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
