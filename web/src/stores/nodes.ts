import { create } from 'zustand'
import type { Node } from '@/lib/types'
import { meshApi } from '@/lib/api'

interface NodesState {
  nodes: Node[]
  loading: boolean
  fetch: () => Promise<void>
  getNode: (id: string) => Node | undefined
  removeNode: (id: string) => Promise<void>
}

export const useNodesStore = create<NodesState>((set, get) => ({
  nodes: [],
  loading: false,

  fetch: async () => {
    set({ loading: true })
    try {
      const nodes = await meshApi.nodes.list()
      set({ nodes })
    } finally {
      set({ loading: false })
    }
  },

  getNode: (id: string) => {
    return get().nodes.find((n) => n.id === id)
  },

  removeNode: async (id: string) => {
    await meshApi.nodes.remove(id)
    await get().fetch()
  },
}))
