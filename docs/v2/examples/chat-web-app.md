# Build a chat web app

Build a single kdeps project that serves both a chat API
and the static web page that talks to it. The browser posts messages to
`/api/v1/chat`. kdeps runs an LLM and returns the reply.

One workflow serves the API route and the web page:

```yaml
# workflow.yaml (part)
deploy:
  apiServer:
    routes:
      - path: /api/v1/chat
        methods: [POST]
        public: true              # browser calls this without a token
  webServer:
    routes:
      - path: /
        serverType: static
        publicPath: ./public      # the frontend
```

## Read next

- [Full walkthrough](/reference/examples/chat-web-app) - every step, file and command
- [Static site tutorial](/examples/static-site) - web server mode on its own
- [LLM resource](/workflow/resources/llm) - `scenario:`, streaming, JSON mode
