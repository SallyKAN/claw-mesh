'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  LayoutDashboard,
  Server,
  GitBranch,
  Terminal,
  RefreshCw,
  Plus,
  ChevronLeft,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useUIStore } from '@/stores/ui'

const navItems = [
  { label: 'MESH', href: '/', icon: LayoutDashboard },
  { label: 'NODES', href: '/nodes', icon: Server },
  { label: 'ROUTE', href: '/routing', icon: GitBranch },
  { label: 'CHAT', href: '/chat', icon: Terminal },
  { label: 'SYNC', href: '/sync', icon: RefreshCw },
]

export function Sidebar() {
  const pathname = usePathname()
  const collapsed = useUIStore((s) => s.sidebarCollapsed)
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)

  return (
    <aside
      className={cn(
        'fixed top-0 left-0 h-screen bg-surface border-r border-border z-40 flex flex-col transition-all duration-200',
        collapsed ? 'w-14' : 'w-48'
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between h-10 px-3 border-b border-border">
        {!collapsed && (
          <span className="text-green text-xs font-bold uppercase tracking-widest">
            claw
          </span>
        )}
        <button
          onClick={toggleSidebar}
          className="text-text-dim hover:text-green transition-colors"
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          <ChevronLeft
            size={16}
            className={cn(
              'transition-transform duration-200',
              collapsed && 'rotate-180'
            )}
          />
        </button>
      </div>

      {/* Navigation */}
      <nav className="flex-1 py-2" role="navigation" aria-label="Main navigation">
        <ul className="space-y-0.5">
          {navItems.map((item) => {
            const isActive =
              item.href === '/'
                ? pathname === '/'
                : pathname.startsWith(item.href)

            return (
              <li key={item.href}>
                <Link
                  href={item.href}
                  className={cn(
                    'flex items-center gap-3 px-3 py-2 text-xs uppercase tracking-widest transition-colors',
                    collapsed && 'justify-center px-0',
                    isActive
                      ? 'text-green border-l-2 border-green bg-surface-2'
                      : 'text-text-dim hover:text-text border-l-2 border-transparent'
                  )}
                  title={collapsed ? item.label : undefined}
                >
                  <item.icon size={16} className="shrink-0" />
                  {!collapsed && <span>{item.label}</span>}
                </Link>
              </li>
            )
          })}
        </ul>
      </nav>

      {/* Bottom: Wizard */}
      <div className="border-t border-border py-2">
        <Link
          href="/wizard"
          className={cn(
            'flex items-center gap-3 px-3 py-2 text-xs uppercase tracking-widest text-amber hover:text-green transition-colors',
            collapsed && 'justify-center px-0'
          )}
          title={collapsed ? 'WIZARD' : undefined}
        >
          <Plus size={16} className="shrink-0" />
          {!collapsed && <span>WIZARD</span>}
        </Link>
      </div>
    </aside>
  )
}
