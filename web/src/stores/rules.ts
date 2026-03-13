import { create } from 'zustand'
import type { RoutingRule, NewRule } from '@/lib/types'
import { meshApi } from '@/lib/api'

interface RulesState {
  rules: RoutingRule[]
  loading: boolean
  fetch: () => Promise<void>
  add: (rule: NewRule) => Promise<void>
  remove: (id: string) => Promise<void>
}

export const useRulesStore = create<RulesState>((set, get) => ({
  rules: [],
  loading: false,

  fetch: async () => {
    set({ loading: true })
    try {
      const rules = await meshApi.rules.list()
      set({ rules })
    } finally {
      set({ loading: false })
    }
  },

  add: async (rule: NewRule) => {
    await meshApi.rules.add(rule)
    await get().fetch()
  },

  remove: async (id: string) => {
    await meshApi.rules.remove(id)
    await get().fetch()
  },
}))
