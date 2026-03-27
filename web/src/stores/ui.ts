import { create } from 'zustand'

interface UIState {
  sidebarCollapsed: boolean
  wizardOpen: boolean
  toggleSidebar: () => void
  toggleWizard: () => void
}

export const useUIStore = create<UIState>((set) => ({
  sidebarCollapsed: false,
  wizardOpen: false,

  toggleSidebar: () => {
    set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed }))
  },

  toggleWizard: () => {
    set((state) => ({ wizardOpen: !state.wizardOpen }))
  },
}))
