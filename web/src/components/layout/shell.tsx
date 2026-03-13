'use client'

import { cn } from '@/lib/utils'
import { useUIStore } from '@/stores/ui'
import { Sidebar } from './sidebar'
import { Topbar } from './topbar'

export function Shell({ children }: { children: React.ReactNode }) {
  const collapsed = useUIStore((s) => s.sidebarCollapsed)

  return (
    <>
      <Topbar />
      <Sidebar />
      <main
        className={cn(
          'pt-14 px-6 pb-6 transition-all duration-200',
          collapsed ? 'ml-14' : 'ml-48'
        )}
      >
        {children}
      </main>
    </>
  )
}
