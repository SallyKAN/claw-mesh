import { create } from 'zustand'
import type { SyncManifest, SyncNodeStatus } from '@/lib/types'
import { meshApi } from '@/lib/api'

interface SyncState {
  manifest: SyncManifest | null
  nodeStatuses: SyncNodeStatus[]
  loading: boolean
  fetchManifest: () => Promise<void>
  fetchStatuses: () => Promise<void>
}

export const useSyncStore = create<SyncState>((set) => ({
  manifest: null,
  nodeStatuses: [],
  loading: false,

  fetchManifest: async () => {
    set({ loading: true })
    try {
      const manifest = await meshApi.sync.manifest()
      set({ manifest })
    } finally {
      set({ loading: false })
    }
  },

  fetchStatuses: async () => {
    set({ loading: true })
    try {
      const nodeStatuses = await meshApi.sync.status()
      set({ nodeStatuses })
    } finally {
      set({ loading: false })
    }
  },
}))
