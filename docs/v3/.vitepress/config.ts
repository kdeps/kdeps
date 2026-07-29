import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'kdeps',
  description: 'AI workflows in YAML — CLI first.',
  appearance: 'force-dark',
  cleanUrls: true,
  lastUpdated: true,

  head: [
    ['link', { rel: 'icon', href: '/favicon.ico', sizes: 'any' }],
    ['link', { rel: 'icon', href: '/favicon-32x32.png', sizes: '32x32', type: 'image/png' }],
    ['link', { rel: 'icon', href: '/favicon-16x16.png', sizes: '16x16', type: 'image/png' }],
    ['link', { rel: 'apple-touch-icon', href: '/apple-touch-icon.png', sizes: '180x180' }],
    ['meta', { name: 'theme-color', content: '#080808' }],
    ['meta', { name: 'og:type', content: 'website' }],
    ['meta', { name: 'og:site_name', content: 'kdeps docs' }],
    ['meta', { name: 'og:title', content: 'kdeps' }],
    ['meta', { name: 'og:description', content: 'AI workflows in YAML — CLI first.' }],
  ],

  themeConfig: {
    logo: '/kdeps-logo.png',
    siteTitle: false,

    nav: [
      { text: 'Quickstart', link: '/quickstart' },
      { text: 'CLI', link: '/cli' },
      { text: 'Workflow', link: '/workflow' },
      { text: 'Deploy', link: '/deploy' },
      { text: 'Registry', link: 'https://kdeps.io' },
      { text: 'GitHub', link: 'https://github.com/kdeps/kdeps' },
      {
        text: 'v3',
        items: [
          { text: 'Changelog', link: 'https://github.com/kdeps/kdeps/releases' },
          { text: 'Archive v2', link: '/v2/' },
          { text: 'Archive v1', link: '/v1/' },
        ],
      },
    ],

    sidebar: [
      {
        text: 'CLI & workflow',
        items: [
          { text: 'Install', link: '/install' },
          { text: 'Quickstart', link: '/quickstart' },
          { text: 'CLI', link: '/cli' },
          { text: 'Workflow mode', link: '/workflow' },
          { text: 'Two modes', link: '/modes' },
        ],
      },
      {
        text: 'Build',
        items: [
          { text: 'Resources', link: '/resources' },
          { text: 'Expressions', link: '/expressions' },
          { text: 'Config', link: '/config' },
          { text: 'Agent mode', link: '/agent' },
        ],
      },
      {
        text: 'Ship',
        items: [
          { text: 'Deploy', link: '/deploy' },
          { text: 'LLM server', link: '/llm-server' },
          { text: 'TLS / HTTPS', link: '/tls' },
          { text: 'Security', link: '/security' },
        ],
      },
      {
        text: 'Reference',
        items: [
          { text: 'Components', link: '/components' },
        ],
      },
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/kdeps/kdeps' },
    ],

    search: {
      provider: 'local',
    },

    editLink: {
      pattern: 'https://github.com/kdeps/kdeps/edit/main/docs/v3/:path',
      text: 'Edit this page',
    },

    footer: {
      message: 'Apache 2.0 · Highly experimental',
      copyright: 'kdeps',
    },
  },
})
