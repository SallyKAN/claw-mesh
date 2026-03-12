import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'claw-mesh',
  description: 'Multi-machine orchestration for OpenClaw',
  base: '/claw-mesh/',

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/claw-mesh/logo.svg' }],
  ],

  themeConfig: {
    nav: [
      { text: 'Guide', link: '/guide/what-is-claw-mesh' },
      { text: 'Reference', link: '/reference/cli' },
      { text: 'GitHub', link: 'https://github.com/SallyKAN/claw-mesh' },
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'Introduction',
          items: [
            { text: 'What is claw-mesh?', link: '/guide/what-is-claw-mesh' },
            { text: 'Quick Start', link: '/guide/quick-start' },
            { text: 'Architecture', link: '/guide/architecture' },
          ],
        },
        {
          text: 'Core Concepts',
          items: [
            { text: 'Nodes & Capabilities', link: '/guide/nodes' },
            { text: 'Routing', link: '/guide/routing' },
            { text: 'Config Seed', link: '/guide/config-seed' },
            { text: 'Security', link: '/guide/security' },
          ],
        },
      ],
      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'CLI', link: '/reference/cli' },
            { text: 'Configuration', link: '/reference/configuration' },
            { text: 'REST API', link: '/reference/api' },
          ],
        },
      ],
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/SallyKAN/claw-mesh' },
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright 2026 claw-mesh contributors',
    },

    search: {
      provider: 'local',
    },
  },
})
