import { defineConfig } from 'vitepress'

/**
 * docs/v3 is the AI Appliances book (synced from ./book/).
 * Agent chapter is ordered before workflow.
 */
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
      { text: 'Start', link: '/chapter-02-getting-started' },
      { text: 'Agent', link: '/chapter-05-agent-mode' },
      { text: 'Workflow', link: '/chapter-04-workflow-mode' },
      { text: 'CLI', link: '/chapter-02-getting-started#install' },
      { text: 'Registry', link: 'https://kdeps.io' },
      { text: 'GitHub', link: 'https://github.com/kdeps/kdeps' },
      {
        text: 'v2.1.11',
        items: [
          { text: 'Changelog', link: 'https://github.com/kdeps/kdeps/releases' },
          { text: 'Release v2.1.11', link: 'https://github.com/kdeps/kdeps/releases/tag/v2.1.11' },
          { text: 'LeanPub', link: 'https://leanpub.com/kdeps' },
          { text: 'Archive v2', link: '/v2/' },
          { text: 'Archive v1', link: '/v1/' },
        ],
      },
    ],

    sidebar: [
      {
        text: 'Start',
        items: [
          { text: 'Preface', link: '/frontmatter' },
          { text: 'Why AI appliances?', link: '/chapter-01-why-ai-appliances' },
          { text: 'Getting started', link: '/chapter-02-getting-started' },
          { text: 'Core concepts', link: '/chapter-03-core-concepts' },
        ],
      },
      {
        text: 'Agent & workflow',
        items: [
          { text: 'Coding agent (agent mode)', link: '/chapter-05-agent-mode' },
          { text: 'Workflow mode', link: '/chapter-04-workflow-mode' },
        ],
      },
      {
        text: 'Build',
        items: [
          { text: 'LLM resources', link: '/chapter-06-llm-resources' },
          { text: 'Data resources', link: '/chapter-07-data-resources' },
          { text: 'Knowledge resources', link: '/chapter-08-knowledge-resources' },
          { text: 'Browser automation', link: '/chapter-09-browser-automation' },
          { text: 'API response & validation', link: '/chapter-10-api-response-and-validation' },
          { text: 'Expressions & data flow', link: '/chapter-11-expressions-and-data-flow' },
          { text: 'Components', link: '/chapter-12-components' },
          { text: 'Agencies', link: '/chapter-13-agencies' },
        ],
      },
      {
        text: 'Config',
        collapsed: true,
        items: [
          { text: 'Workflow configuration', link: '/chapter-14-workflow-configuration' },
          { text: 'Sessions, CORS, routes', link: '/chapter-15-sessions-cors-routes' },
          { text: 'Advanced configuration', link: '/chapter-16-advanced-configuration' },
        ],
      },
      {
        text: 'Ship & patterns',
        collapsed: true,
        items: [
          { text: 'Docker', link: '/chapter-17-docker-deployment' },
          { text: 'Kubernetes', link: '/chapter-18-kubernetes-deployment' },
          { text: 'Standalone binary', link: '/chapter-19-standalone-binary' },
          { text: 'Web server mode', link: '/chapter-20-webserver-mode' },
          { text: 'Validate, debug, develop', link: '/chapter-21-validate-debug-develop' },
          { text: 'Iteration', link: '/chapter-22-iteration' },
          { text: 'Error handling', link: '/chapter-23-error-handling' },
          { text: 'Real-world examples', link: '/chapter-24-real-world-examples' },
          { text: 'Bot & file inputs', link: '/chapter-25-bot-and-file-inputs' },
          { text: 'LLM server appliance', link: '/chapter-26-llm-server-appliance' },
        ],
      },
      {
        text: 'Appendix',
        collapsed: true,
        items: [
          { text: 'Troubleshooting', link: '/appendix-a-troubleshooting' },
          { text: 'Security', link: '/appendix-b-security' },
          { text: 'Testing', link: '/appendix-c-testing' },
          { text: 'About the author', link: '/backmatter' },
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
      message: 'Apache 2.0 · kdeps v2.1.11 · AI Appliances',
      copyright: 'kdeps',
    },
  },
})
