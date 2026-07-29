# Web server mode

Serve UI and API on **one port**.

```text
:16395
├── /api/v1/*  → workflow DAG
├── /app/*     → optional proxy to frontend dev server
└── /*         → static files (publicPath)
```

```yaml
settings:
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]

  webServer:
    publicPath: "./public"     # production static assets
    # Dev proxy (optional):
    # command: "npm run dev"
    # appPort: 3000
    # appPathPrefix: "/app"
    # workDir: "./frontend"
```

## Static (production)

Build your SPA into `public/`, package with the workflow:

```text
my-agent/
├── workflow.yaml
├── resources/
└── public/
    ├── index.html
    └── …
```

## Subprocess proxy (dev)

kdeps starts `command`, waits for `appPort`, proxies `appPathPrefix` there. API routes still hit the DAG. Use `kdeps run --dev` for YAML reload; the frontend HMR stays on its own tooling.

## CORS

If the UI is on another origin, configure CORS under `apiServer` (see [Config](/config)). Same-port static / proxy often needs less CORS wiring.

[Deploy](/deploy) · [Workflow](/workflow) · [Security](/security).
