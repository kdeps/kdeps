import { defineConfig } from 'vitepress'

/** Book chapters (from ./book via docs/v3/book). Agent before workflow. */
const bookStart = [
  { text: 'Preface', link: '/book/frontmatter' },
  { text: 'Why AI appliances?', link: '/book/chapter-01-why-ai-appliances' },
  { text: 'Getting started', link: '/book/chapter-02-getting-started' },
  { text: 'Core concepts', link: '/book/chapter-03-core-concepts' },
]

const bookAgentFirst = [
  { text: 'Coding agent (agent mode)', link: '/book/chapter-05-agent-mode' },
  { text: 'Workflow mode', link: '/book/chapter-04-workflow-mode' },
]

const bookResources = [
  { text: 'LLM resources', link: '/book/chapter-06-llm-resources' },
  { text: 'Data resources', link: '/book/chapter-07-data-resources' },
  { text: 'Knowledge resources', link: '/book/chapter-08-knowledge-resources' },
  { text: 'Browser automation', link: '/book/chapter-09-browser-automation' },
  { text: 'API response & validation', link: '/book/chapter-10-api-response-and-validation' },
  { text: 'Expressions & data flow', link: '/book/chapter-11-expressions-and-data-flow' },
  { text: 'Components', link: '/book/chapter-12-components' },
  { text: 'Agencies', link: '/book/chapter-13-agencies' },
]

const bookConfig = [
  { text: 'Workflow configuration', link: '/book/chapter-14-workflow-configuration' },
  { text: 'Sessions, CORS, routes', link: '/book/chapter-15-sessions-cors-routes' },
  { text: 'Advanced configuration', link: '/book/chapter-16-advanced-configuration' },
]

const bookShip = [
  { text: 'Docker deployment', link: '/book/chapter-17-docker-deployment' },
  { text: 'Kubernetes deployment', link: '/book/chapter-18-kubernetes-deployment' },
  { text: 'Standalone binary', link: '/book/chapter-19-standalone-binary' },
  { text: 'Web server mode', link: '/book/chapter-20-webserver-mode' },
  { text: 'Validate, debug, develop', link: '/book/chapter-21-validate-debug-develop' },
  { text: 'Iteration (items & loop)', link: '/book/chapter-22-iteration' },
  { text: 'Error handling (onError)', link: '/book/chapter-23-error-handling' },
  { text: 'Real-world examples', link: '/book/chapter-24-real-world-examples' },
  { text: 'Bot & file inputs', link: '/book/chapter-25-bot-and-file-inputs' },
  { text: 'LLM server appliance', link: '/book/chapter-26-llm-server-appliance' },
]

const bookAppendix = [
  { text: 'Troubleshooting', link: '/book/appendix-a-troubleshooting' },
  { text: 'Security', link: '/book/appendix-b-security' },
  { text: 'Testing', link: '/book/appendix-c-testing' },
  { text: 'About the author', link: '/book/backmatter' },
]

export default defineConfig({
  title: 'kdeps',
  description: 'AI Appliances — coding CLI agent, workflows, and agencies in YAML.',
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
    ['meta', { name: 'og:title', content: 'kdeps — AI Appliances' }],
    ['meta', { name: 'og:description', content: 'Coding CLI agent, then workflows and agencies in YAML.' }],
  ],

  themeConfig: {
    logo: '/kdeps-logo.png',
    siteTitle: false,

    nav: [
      { text: 'Start', link: '/quickstart' },
      { text: 'Agent', link: '/book/chapter-05-agent-mode' },
      { text: 'CLI', link: '/cli' },
      { text: 'Book', link: '/book/frontmatter' },
      { text: 'Registry', link: 'https://kdeps.io' },
      { text: 'GitHub', link: 'https://github.com/kdeps/kdeps' },
      {
        text: 'v2.1.11',
        items: [
          { text: 'Changelog', link: 'https://github.com/kdeps/kdeps/releases' },
          { text: 'Release v2.1.11', link: 'https://github.com/kdeps/kdeps/releases/tag/v2.1.11' },
          { text: 'LeanPub book', link: 'https://leanpub.com/kdeps' },
          { text: 'Archive v2 docs', link: '/v2/' },
          { text: 'Archive v1 docs', link: '/v1/' },
        ],
      },
    ],

    sidebar: [
      {
        text: 'Quick reference',
        items: [
          { text: 'Install', link: '/install' },
          { text: 'Quickstart', link: '/quickstart' },
          { text: 'CLI', link: '/cli' },
          { text: 'Two modes', link: '/modes' },
        ],
      },
      {
        text: 'Book — start',
        collapsed: false,
        items: bookStart,
      },
      {
        text: 'Book — agent & workflow',
        collapsed: false,
        items: bookAgentFirst,
      },
      {
        text: 'Book — build',
        collapsed: false,
        items: bookResources,
      },
      {
        text: 'Book — config',
        collapsed: true,
        items: bookConfig,
      },
      {
        text: 'Book — ship & patterns',
        collapsed: true,
        items: bookShip,
      },
      {
        text: 'Book — appendix',
        collapsed: true,
        items: bookAppendix,
      },
      {
        text: 'Quick cards',
        collapsed: true,
        items: [
          { text: 'Coding agent (short)', link: '/agent' },
          { text: 'Workflow (short)', link: '/workflow' },
          { text: 'Resources', link: '/resources' },
          { text: 'LLM chat', link: '/llm' },
          { text: 'Agencies', link: '/agencies' },
          { text: 'Inputs', link: '/inputs' },
          { text: 'Iteration', link: '/iteration' },
          { text: 'Errors', link: '/errors' },
          { text: 'Config', link: '/config' },
          { text: 'Web server', link: '/webserver' },
          { text: 'Deploy', link: '/deploy' },
          { text: 'LLM server', link: '/llm-server' },
          { text: 'TLS', link: '/tls' },
          { text: 'Security', link: '/security' },
          { text: 'Debug', link: '/debug' },
          { text: 'Components', link: '/components' },
          { text: 'Expressions', link: '/expressions' },
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
      message: 'Apache 2.0 · kdeps v2.1.11 · Book: AI Appliances',
      copyright: 'kdeps',
    },
  },
})
