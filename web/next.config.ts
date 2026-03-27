import type { NextConfig } from 'next'

const config: NextConfig = {
  output: 'export',
  distDir: 'dist',
  trailingSlash: true,
  images: { unoptimized: true },
  async rewrites() {
    return [
      { source: '/api/:path*', destination: 'http://localhost:9180/api/:path*' },
      { source: '/healthz', destination: 'http://localhost:9180/healthz' },
    ]
  },
}

export default config
