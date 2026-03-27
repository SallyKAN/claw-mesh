import type { Metadata } from 'next'
import { Shell } from '@/components/layout/shell'
import './globals.css'

export const metadata: Metadata = {
  title: 'claw-mesh',
  description: 'Multi-Gateway orchestrator for OpenClaw',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en">
      <body className="bg-bg text-text antialiased">
        <Shell>{children}</Shell>
      </body>
    </html>
  )
}
