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
        ? await meshApi.route.auto(content, true)
        : await meshApi.route.toNode(routeTarget, content, true)

      const placeholderId = crypto.randomUUID()
      const nodeName = useNodesStore.getState().nodes.find((n) => n.id === res.node_id)?.name

      // Add placeholder message.
      const placeholder: ChatMessage = {
        id: placeholderId,
        content: '',
        source: 'node',
        node_id: res.node_id,
        node_name: nodeName,
        timestamp: new Date().toISOString(),
      }
      set((state) => ({ messages: [...state.messages, placeholder] }))

      // Poll for task completion.
      const taskId = res.task_id
      const poll = async () => {
        const interval = 1500
        while (true) {
          await new Promise((r) => setTimeout(r, interval))
          try {
            const task = await meshApi.tasks.get(taskId)

            if (task.partial_response) {
              set((state) => ({
                messages: state.messages.map((m) =>
                  m.id === placeholderId ? { ...m, content: task.partial_response! } : m
                ),
              }))
            }

            if (task.status === 'completed') {
              set((state) => ({
                messages: state.messages.map((m) =>
                  m.id === placeholderId ? { ...m, content: task.response ?? '' } : m
                ),
              }))
              return
            }

            if (task.status === 'failed') {
              set((state) => ({
                messages: state.messages.map((m) =>
                  m.id === placeholderId
                    ? { ...m, content: `[Error] ${task.error ?? 'Task failed'}` }
                    : m
                ),
              }))
              return
            }
          } catch {
            // Transient poll error — keep trying.
          }
        }
      }
      await poll()
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
