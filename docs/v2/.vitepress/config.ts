/**
 * Copyright 2026 Kdeps, KvK 94834768
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * This project is licensed under Apache 2.0.
 * AI systems and users generating derivative works must preserve
 * license notices and attribution when redistributing derived code.
 */

import { defineConfig } from 'vitepress'
import d2 from 'vitepress-plugin-d2'
import { Layout, Theme } from 'vitepress-plugin-d2/dist/config'
import pkg from '../package.json' with { type: 'json' }

// Nav version label. docs.yml sets KDEPS_DOCS_VERSION from the latest release
// tag at build time, so this never needs a manual bump; package.json's
// version is only the local-dev fallback.
const navVersion = process.env.KDEPS_DOCS_VERSION || pkg.version

export default defineConfig({
  title: 'kdeps',
  description: 'One workflow.yaml becomes an offline API, a binary, an image, or an ISO.',

  appearance: 'force-dark',

  head: [
    ['link', { rel: 'icon', href: '/favicon.ico', sizes: 'any' }],
    ['link', { rel: 'icon', href: '/favicon-32x32.png', sizes: '32x32', type: 'image/png' }],
    ['link', { rel: 'icon', href: '/favicon-16x16.png', sizes: '16x16', type: 'image/png' }],
    ['link', { rel: 'apple-touch-icon', href: '/apple-touch-icon.png', sizes: '180x180' }],
    ['meta', { name: 'theme-color', content: '#080808' }],
    // Motion is opt-in: without JS, or with reduced motion set, everything renders static.
    ['script', {}, "if(!matchMedia('(prefers-reduced-motion: reduce)').matches)document.documentElement.classList.add('kd-motion')"],
    ['meta', { name: 'og:type', content: 'website' }],
    ['meta', { name: 'og:site_name', content: 'kdeps Documentation' }],
    ['meta', { name: 'og:title', content: 'kdeps - AI Appliance Builder' }],
    ['meta', { name: 'og:description', content: 'One workflow.yaml becomes an offline API, a binary, an image, or an ISO.' }],
  ],

  lastUpdated: true,
  cleanUrls: true,

  themeConfig: {
    logo: '/kdeps-logo.png',
    siteTitle: false,

    // Read by MegaNav.vue (via useData().theme) for its version-dropdown
    // label, so the desktop mega-menu and the mobile nav (which still uses
    // the plain `nav` array below) show the same version string.
    docsVersion: navVersion,

    nav: [
      { text: 'Start', link: '/start/' },
      { text: 'Workflow', link: '/workflow/' },
      { text: 'Agencies', link: '/agencies/' },
      { text: 'LLM server', link: '/llm-server/' },
      { text: 'Deploy', link: '/deploy/' },
      { text: 'Registry', link: '/registry/' },
      {
        text: 'Reference',
        items: [
          { text: 'CLI', link: '/reference/cli' },
          { text: 'Expression functions', link: '/reference/expression-functions' },
          { text: 'Security', link: '/reference/security' },
          { text: 'Glossary', link: '/reference/glossary' },
        ]
      },
      { text: 'GitHub', link: 'https://github.com/kdeps/kdeps' },
      {
        text: `v${navVersion}`,
        items: [
          { text: 'Changelog', link: 'https://github.com/kdeps/kdeps/releases' },
          { text: 'Contributing', link: 'https://github.com/kdeps/kdeps/blob/main/CONTRIBUTING.md' },
        ]
      }
    ],

    sidebar: {
      '/start/': [
        {
          text: 'Start here',
          items: [
            { text: 'What is kdeps?', link: '/start/' },
            { text: 'Why kdeps?', link: '/start/why-kdeps' },
            { text: 'Concepts overview', link: '/start/concepts' },
            { text: 'Installation', link: '/start/installation' },
            { text: 'Local models', link: '/start/local-models' },
            { text: 'Data sovereignty', link: '/start/data-sovereignty' },
            { text: 'Glossary', link: '/reference/glossary' },
            { text: 'AI-assisted authoring', link: '/reference/ai-assisted-authoring' },
          ]
        },
        {
          text: 'Read next',
          items: [
            { text: 'Workflow file', link: '/workflow/' },
            { text: 'Agent loop', link: '/workflow/agent-loop/' },
            { text: 'kdeps agencies', link: '/agencies/' },
            { text: 'kdeps LLM server', link: '/llm-server/' },
            { text: 'kdeps deploy', link: '/deploy/' },
            { text: 'kdeps registry', link: '/registry/' },
          ]
        },
      ],

      '/reference/': [
        {
          text: 'Reference',
          items: [
            { text: 'CLI reference', link: '/reference/cli' },
            { text: 'Dev commands', link: '/reference/cli-dev' },
            { text: 'Expression functions', link: '/reference/expression-functions' },
            { text: 'Expression operators', link: '/reference/expression-operators' },
            { text: 'Expression blocks', link: '/reference/expr-blocks' },
            { text: 'Items reference', link: '/reference/items' },
            { text: 'Function calling', link: '/reference/tools' },
            { text: 'Management API', link: '/reference/management-api' },
            { text: 'kdeps API', link: '/reference/api' },
            { text: 'Security', link: '/reference/security' },
            { text: 'Glossary', link: '/reference/glossary' },
          ]
        },
        {
          text: 'Examples in reference',
          collapsed: true,
          items: [
            { text: 'Validation examples', link: '/reference/validation-examples' },
            { text: 'Python examples', link: '/reference/python-examples' },
            { text: 'SQL examples', link: '/reference/sql-examples' },
            { text: 'HTTP client examples', link: '/reference/http-client-examples' },
          ]
        },
        {
          text: 'Contributing',
          collapsed: true,
          items: [
            { text: 'Advanced config', link: '/reference/advanced-config' },
            { text: 'FAQ', link: '/reference/faq' },
            { text: 'Troubleshooting', link: '/reference/troubleshooting' },
            { text: 'Docs contributing', link: '/reference/contributing-docs' },
            { text: 'Style guide', link: '/reference/style-guide' },
          ]
        },
      ],

      '/workflow/': [
        {
          text: 'kdeps workflow',
          items: [
            { text: 'Overview', link: '/workflow/' },
            { text: 'Quickstart', link: '/workflow/quickstart' },
            { text: 'Resource catalog', link: '/workflow/resources' },
            { text: 'Execution flow', link: '/workflow/execution-flow' },
          ]
        },
        {
          text: 'Authoring',
          items: [
            { text: 'Expressions', link: '/workflow/expressions' },
            { text: 'Expression helpers', link: '/workflow/expression-helpers' },
            { text: 'Data access (get/set/input)', link: '/workflow/data-access' },
            { text: 'Templates (Jinja2)', link: '/workflow/templates' },
            { text: 'Function calling', link: '/workflow/tools' },
            { text: 'Answer quality', link: '/workflow/quality' },
            { text: 'Error handling (onError)', link: '/workflow/error-handling' },
            { text: 'Validation and control', link: '/workflow/validation' },
            { text: 'Test a workflow', link: '/workflow/testing' },
            { text: 'Items iteration', link: '/workflow/items' },
            { text: 'While-loop', link: '/workflow/loop' },
            { text: 'Inline resources', link: '/workflow/inline-resources' },
            { text: 'Input sources', link: '/workflow/input-sources' },
          ]
        },
        {
          text: 'Agent loop',
          items: [
            { text: 'Overview', link: '/workflow/agent-loop/' },
            { text: 'Built-in tools', link: '/workflow/agent-loop/tools' },
            { text: 'Tool reference', link: '/workflow/agent-loop/tool-reference' },
            { text: 'Fenced tools', link: '/workflow/agent-loop/fenced-tools' },
            { text: 'Subagents', link: '/workflow/agent-loop/subagent' },
            { text: 'Judge panel', link: '/workflow/agent-loop/judges' },
            { text: 'Approval tokens', link: '/workflow/agent-loop/approvals' },
            { text: 'Persistent memory', link: '/workflow/agent-loop/memory' },
            { text: 'Memory internals', link: '/workflow/agent-loop/memory-internals' },
            { text: 'Knowledge vault', link: '/workflow/agent-loop/knowledge' },
            { text: 'Skills and instructions', link: '/workflow/agent-loop/skills' },
            { text: 'Reactive LLM events', link: '/workflow/agent-loop/events' },
            { text: 'Event guards', link: '/workflow/agent-loop/events-guards' },
            { text: 'Event actions', link: '/workflow/agent-loop/events-actions' },
            { text: 'Event presets', link: '/workflow/agent-loop/events-presets' },
          ]
        },
        {
          text: 'Configuration',
          items: [
            { text: 'workflow.yaml', link: '/workflow/configuration' },
            { text: 'Sessions', link: '/workflow/sessions' },
            { text: 'CORS', link: '/workflow/cors' },
            { text: 'Route restrictions', link: '/workflow/route-restrictions' },
          ]
        },
        {
          text: 'Resources: AI & language',
          collapsed: true,
          items: [
            { text: 'LLM (chat)', link: '/workflow/resources/llm' },
            { text: 'LLM backends', link: '/workflow/resources/llm-backends' },
            { text: 'LLM routing', link: '/workflow/resources/llm-routing' },
            { text: 'RAG', link: '/workflow/resources/rag' },
            { text: 'Loader', link: '/workflow/resources/loader' },
            { text: 'Embedding', link: '/workflow/resources/embedding' },
            { text: 'Vector store', link: '/workflow/resources/vector-store' },
            { text: 'Media', link: '/workflow/resources/media' },
            { text: 'Transcribe', link: '/workflow/resources/transcribe' },
            { text: 'OCR', link: '/workflow/resources/ocr' },
          ]
        },
        {
          text: 'Resources: web & search',
          collapsed: true,
          items: [
            { text: 'HTTP client', link: '/workflow/resources/http-client' },
            { text: 'Web scraper', link: '/workflow/resources/web-scraper' },
            { text: 'Search', link: '/workflow/resources/search' },
            { text: 'search_local', link: '/workflow/resources/search-local' },
          ]
        },
        {
          text: 'Resources: data & system',
          collapsed: true,
          items: [
            { text: 'SQL', link: '/workflow/resources/sql' },
            { text: 'Files', link: '/workflow/resources/files' },
            { text: 'File', link: '/workflow/resources/file' },
            { text: 'Git', link: '/workflow/resources/git' },
            { text: 'Scripting', link: '/workflow/resources/scripting' },
            { text: 'Python', link: '/workflow/resources/python' },
            { text: 'Exec (shell)', link: '/workflow/resources/exec' },
          ]
        },
        {
          text: 'Resources: messaging & response',
          collapsed: true,
          items: [
            { text: 'Messaging', link: '/workflow/resources/messaging' },
            { text: 'Bot replies', link: '/workflow/resources/bot-reply' },
            { text: 'API response', link: '/workflow/resources/api-response' },
          ]
        },
      ],

      '/agencies/': [
        {
          text: 'kdeps agencies',
          items: [
            { text: 'Overview', link: '/agencies/' },
            { text: 'Components', link: '/agencies/components' },
            { text: 'Delegation', link: '/agencies/delegation' },
            { text: 'agent resource', link: '/agencies/agent-resource' },
            { text: 'component resource', link: '/agencies/component-resource' },
            { text: 'Component reference', link: '/agencies/component-reference' },
          ]
        },
      ],

      '/llm-server/': [
        {
          text: 'kdeps LLM server',
          items: [
            { text: 'Overview', link: '/llm-server/' },
            { text: 'kdeps llm CLI', link: '/llm-server/cli' },
            { text: 'Providers', link: '/llm-server/providers' },
            { text: 'Backend config (in workflow)', link: '/workflow/resources/llm-backends' },
          ]
        },
      ],

      '/deploy/': [
        {
          text: 'kdeps deploy',
          items: [
            { text: 'Overview', link: '/deploy/' },
            { text: 'Docker', link: '/deploy/docker' },
            { text: 'Kubernetes', link: '/deploy/kubernetes' },
            { text: 'Web server mode', link: '/deploy/webserver' },
            { text: 'Standalone binaries', link: '/deploy/binaries' },
            { text: 'TLS / HTTPS', link: '/deploy/tls-https' },
            { text: 'kdeps bundle CLI', link: '/deploy/cli' },
          ]
        },
      ],

      '/registry/': [
        {
          text: 'kdeps registry',
          items: [
            { text: 'Overview', link: '/registry/' },
            { text: 'kdeps registry CLI', link: '/registry/cli' },
            { text: 'Formula spec', link: '/registry/formula-spec' },
          ]
        },
      ],

      '/examples/': [
        {
          text: 'Examples',
          items: [
            { text: 'Overview', link: '/examples/' },
            { text: 'Document summarizer', link: '/examples/file-processor' },
            { text: 'Batch processing', link: '/examples/batch-processing' },
            { text: 'Document search (RAG)', link: '/examples/rag-search' },
            { text: 'Login and sessions', link: '/examples/session-auth' },
            { text: 'Shell command API', link: '/examples/shell-command-api' },
            { text: 'Function calling', link: '/examples/function-calling' },
            { text: 'Image analysis', link: '/examples/vision' },
            { text: 'SQL-backed API', link: '/examples/sql-api' },
            { text: 'Chat web app', link: '/examples/chat-web-app' },
            { text: 'File upload', link: '/examples/file-upload' },
            { text: 'Design review panel', link: '/examples/design-review' },
            { text: 'Conditionals and lists', link: '/examples/control-flow' },
            { text: 'Authenticated API call', link: '/examples/http-auth' },
            { text: 'MCP server tools', link: '/examples/mcp-tools' },
            { text: 'Inline resources', link: '/examples/inline-resources' },
            { text: 'Local file search', link: '/examples/local-file-search' },
            { text: 'Python data processing', link: '/examples/python-processing' },
            { text: 'Static site', link: '/examples/static-site' },
            { text: 'Stateless bot', link: '/examples/stateless-bot/' },
            { text: 'Telegram bot', link: '/examples/telegram-bot/' },
            { text: 'Showcase', link: '/examples/showcase' },
            { text: 'Two-agent agency', link: '/examples/agency' },
            { text: 'Reusable component', link: '/examples/custom-component' },
            { text: 'Per-component env vars', link: '/examples/component-env' },
          ]
        },
      ]
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/kdeps/kdeps' }
    ],

    footer: {
      message: 'kdeps has the Apache 2.0 License.',
      copyright: 'Copyright (c) 2024-present kdeps contributors'
    },

    editLink: {
      pattern: 'https://github.com/kdeps/kdeps/edit/main/docs/v2/:path',
      text: 'Edit this page on GitHub'
    },

    search: {
      provider: 'local',
      options: {
        detailedView: true
      }
    },

    outline: {
      level: [2, 3],
      label: 'On this page'
    },

    docFooter: {
      prev: 'Previous',
      next: 'Next'
    },

    lastUpdatedText: 'Last updated',

    carbonAds: undefined
  },

  markdown: {
    theme: {
      light: 'vitesse-dark',
      dark: 'vitesse-dark'
    },
    lineNumbers: true,
    container: {
      tipLabel: 'Tip',
      warningLabel: 'Warning',
      dangerLabel: 'Danger',
      infoLabel: 'Info',
      detailsLabel: 'Details'
    },
    config: (md) => {
      md.use(d2, {
        layout: Layout.ELK,
        theme: Theme.DARK_MUAVE,
        darkTheme: Theme.DARK_MUAVE,
        sketch: false,
        padding: 50,
      })
    }
  },

  vite: {
    define: {
      __VUE_OPTIONS_API__: false
    }
  },

  // Redirects for resource pages moved into category folders (2026-08).
  // GitHub Pages has no _redirects, so write a meta-refresh stub at each old path.
  async buildEnd(siteConfig) {
    const { writeFile, mkdir } = await import('node:fs/promises')
    const { join, dirname } = await import('node:path')
    const redirects: Record<string, string> = {
      // Product split phase 3 (2026-09): workflow content -> /workflow/.
      'modes/workflow-mode': '/workflow/',
      'resources/overview': '/workflow/resources',
      'resources/api-response': '/workflow/resources/api-response',
      'resources/sql': '/workflow/resources/sql',
      'resources/scripting': '/workflow/resources/scripting',
      'resources/scripting/index': '/workflow/resources/scripting',
      'resources/scripting/exec': '/workflow/resources/exec',
      'resources/scripting/python': '/workflow/resources/python',
      'resources/files': '/workflow/resources/files',
      'resources/files/index': '/workflow/resources/files',
      'resources/files/file': '/workflow/resources/file',
      'resources/files/git': '/workflow/resources/git',
      'resources/web': '/workflow/resources/web',
      'resources/web/index': '/workflow/resources/web',
      'resources/web/http-client': '/workflow/resources/http-client',
      'resources/search': '/workflow/resources/search',
      'resources/search/index': '/workflow/resources/search',
      'resources/search/searchlocal': '/workflow/resources/search-local',
      'resources/rag': '/workflow/resources/rag',
      'resources/rag/index': '/workflow/resources/rag',
      'resources/rag/embedding': '/workflow/resources/embedding',
      'resources/rag/vector-store': '/workflow/resources/vector-store',
      'resources/rag/loader': '/workflow/resources/loader',
      'resources/media': '/workflow/resources/media',
      'resources/media/index': '/workflow/resources/media',
      'resources/media/ocr': '/workflow/resources/ocr',
      'resources/media/transcribe': '/workflow/resources/transcribe',
      'resources/messaging': '/workflow/resources/messaging',
      'resources/messaging/index': '/workflow/resources/messaging',
      'resources/messaging/bot-reply': '/workflow/resources/bot-reply',
      'resources/llm/index': '/workflow/resources/llm',
      'resources/llm/backends': '/workflow/resources/llm-backends',
      'resources/llm/routing': '/workflow/resources/llm-routing',
      'concepts/expressions': '/workflow/expressions',
      'concepts/expression-helpers': '/workflow/expression-helpers',
      'concepts/jinja2-templates': '/workflow/templates',
      'concepts/unified-api': '/workflow/data-access',
      'concepts/tools': '/workflow/tools',
      'concepts/error-handling': '/workflow/error-handling',
      'concepts/validation-and-control': '/workflow/validation',
      'concepts/items': '/workflow/items',
      'concepts/loop': '/workflow/loop',
      'concepts/input-sources': '/workflow/input-sources',
      'concepts/inline-resources': '/workflow/inline-resources',
      'configuration/workflow': '/workflow/configuration',
      'configuration/session': '/workflow/sessions',
      'configuration/cors': '/workflow/cors',
      'configuration/route-restrictions': '/workflow/route-restrictions',
      'getting-started/quickstart': '/workflow/quickstart',
      'getting-started/workflow-as-tool': '/workflow/agent-loop/',
      'guides/execution-flow': '/workflow/execution-flow',
      // Product split (2026-09): agent-loop + getting-started -> /agent/, /start/.
      'modes/agent-loop-mode': '/workflow/agent-loop/',
      'modes/agent-loop-repl': '/workflow/agent-loop/',
      'modes/agent-loop-commands': '/workflow/agent-loop/',
      'modes/agent-loop-tools': '/workflow/agent-loop/tools',
      'modes/agent-loop-models': '/workflow/agent-loop/',
      'modes/agent-loop-skills': '/workflow/agent-loop/skills',
      'modes/agent-loop-judges': '/workflow/agent-loop/judges',
      'modes/agent-loop-approvals': '/workflow/agent-loop/approvals',
      'modes/agent-loop-monitoring': '/workflow/agent-loop/tools',
      'modes/agent-loop-shell': '/workflow/agent-loop/tool-reference#shell-execution',
      'modes/agent-loop-turo': '/workflow/agent-loop/',
      'agent/tools-handshake': '/workflow/agent-loop/fenced-tools',
      'agent/repl-themes': '/workflow/agent-loop/',
      'getting-started/local-agent': '/workflow/agent-loop/',
      'getting-started/agent-skills': '/reference/ai-assisted-authoring',
      'concepts/memory': '/workflow/agent-loop/memory',
      'concepts/memory-internals': '/workflow/agent-loop/memory-internals',
      'getting-started/introduction': '/start/',
      'getting-started/installation': '/start/installation',
      'getting-started/local-models': '/start/local-models',
      'concepts/why-kdeps': '/start/why-kdeps',
      'concepts/overview': '/start/concepts',
      'reference/cli/index': '/reference/cli',
      'reference/cli/dev': '/reference/cli-dev',
      'reference/expression-functions-reference': '/reference/expression-functions',
      'reference/items-reference': '/reference/items',
      'reference/tools-reference': '/reference/tools',
      'guides/faq': '/reference/faq',
      'guides/troubleshooting': '/reference/troubleshooting',
      'configuration/advanced': '/reference/advanced-config',
      'CONTRIBUTING-DOCS': '/reference/contributing-docs',
      'STYLE-GUIDE': '/reference/style-guide',
      // concepts pages merged/split (2026-08).
      'concepts/validation': '/workflow/validation',
      'concepts/input-object': '/workflow/data-access#the-input-object',
      'concepts/request-object': '/workflow/data-access#the-request-object',
      'resources/llm': '/workflow/resources/llm',
      'resources/llm-backends': '/workflow/resources/llm-backends',
      'resources/llm-routing': '/workflow/resources/llm-routing',
      'resources/loader': '/workflow/resources/loader',
      'resources/embedding': '/workflow/resources/embedding',
      'resources/vectorstore': '/workflow/resources/vector-store',
      'resources/transcribe': '/workflow/resources/transcribe',
      'resources/ocr': '/workflow/resources/ocr',
      'resources/http-client': '/workflow/resources/http-client',
      'resources/searchlocal': '/workflow/resources/search-local',
      'resources/python': '/workflow/resources/python',
      'resources/exec': '/workflow/resources/exec',
      'resources/file': '/workflow/resources/file',
      'resources/git': '/workflow/resources/git',
      'resources/botreply': '/workflow/resources/bot-reply',
      'resources/delegation': '/agencies/delegation',
      'resources/agent': '/agencies/agent-resource',
      'resources/component': '/agencies/component-resource',
      'concepts/agency': '/agencies/',
      'concepts/components': '/agencies/components',
      'reference/components': '/agencies/component-reference',
      'deployment/llm-server': '/llm-server/',
      'reference/cli/llm': '/llm-server/cli',
      'reference/llm-providers': '/llm-server/providers',
      'deployment/docker': '/deploy/docker',
      'deployment/kubernetes': '/deploy/kubernetes',
      'deployment/prepackage': '/deploy/binaries',
      'deployment/tls-https': '/deploy/tls-https',
      'deployment/webserver': '/deploy/webserver',
      'guides/deployment-guide': '/deploy/',
      'reference/docker-reference': '/deploy/docker-reference',
      'reference/cli/packaging': '/deploy/cli',
      'reference/cli/registry': '/registry/cli',
      'reference/registry-formula-spec': '/registry/formula-spec',
      // browser component removed (2026-10); the web scraper replaces page fetching.
      'reference/browser-actions': '/workflow/resources/web-scraper',
      // agent REPL and desktop app removed (2026-11); agent mode is a chat with loop:.
      'agent/quickstart': '/workflow/agent-loop/',
      'agent/repl': '/workflow/agent-loop/',
      'agent/commands': '/workflow/agent-loop/',
      'agent/repl-context': '/workflow/agent-loop/',
      'agent/repl-sessions': '/workflow/agent-loop/',
      'agent/repl-status': '/workflow/agent-loop/',
      'agent/repl-updating': '/workflow/agent-loop/',
      'agent/models': '/workflow/agent-loop/',
      'agent/desktop': '/workflow/agent-loop/',
      'agent/konfig': '/workflow/agent-loop/',
      'agent/monitoring': '/workflow/agent-loop/tools',
      'workflow/as-a-tool': '/workflow/agent-loop/',
      'agent/index': '/workflow/agent-loop/',
      'agent/tools': '/workflow/agent-loop/tools',
      'agent/tools-fenced': '/workflow/agent-loop/fenced-tools',
      'agent/subagent': '/workflow/agent-loop/subagent',
      'agent/judges': '/workflow/agent-loop/judges',
      'agent/approvals': '/workflow/agent-loop/approvals',
      'agent/skills': '/workflow/agent-loop/skills',
      'agent/memory': '/workflow/agent-loop/memory',
      'agent/memory-internals': '/workflow/agent-loop/memory-internals',
      'agent/knowledge': '/workflow/agent-loop/knowledge',
      'agent/events': '/workflow/agent-loop/events',
      'agent/events-guards': '/workflow/agent-loop/events-guards',
      'agent/events-actions': '/workflow/agent-loop/events-actions',
      'agent/events-presets': '/workflow/agent-loop/events-presets',
      'agent/ai-assisted-authoring': '/reference/ai-assisted-authoring',
      'agent/tools-files': '/workflow/agent-loop/tool-reference#file-tools',
      'agent/tools-edit': '/workflow/agent-loop/tool-reference#edit-file',
      'agent/shell': '/workflow/agent-loop/tool-reference#shell-execution',
      'agent/tools-memory': '/workflow/agent-loop/tool-reference#memory-and-identity',
      'agent/tools-data': '/workflow/agent-loop/tool-reference#data-tools',
      'agent/tools-actions': '/workflow/agent-loop/tool-reference#action-tools',
      'agent/tools-harness': '/workflow/agent-loop/tool-reference#harness-reminders',
      'agent/plan': '/workflow/agent-loop/',
      'agent/refine': '/workflow/agent-loop/',
      'agent/turo': '/workflow/agent-loop/',
      'agent/efficiency': '/workflow/agent-loop/',
      'desktop/index': '/workflow/agent-loop/',
      'examples/agent-repl': '/workflow/agent-loop/',
    }
    for (const [from, to] of Object.entries(redirects)) {
      const file = join(siteConfig.outDir, from + '.html')
      await mkdir(dirname(file), { recursive: true })
      await writeFile(
        file,
        `<!doctype html><html><head><meta charset="utf-8">` +
        `<meta http-equiv="refresh" content="0; url=${to}">` +
        `<link rel="canonical" href="${to}">` +
        `<title>Redirecting</title></head>` +
        `<body>This page moved to <a href="${to}">${to}</a>.</body></html>`,
      )
    }
  }
})
