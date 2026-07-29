# Chapter 20: WebServer Mode

`webServer:` serves a frontend application alongside your agent API. Requests hit a single port. A path prefix determines whether they go to the API pipeline, the frontend app, or static files.

## Why WebServer Mode

Most AI agents need a user interface. Without WebServer mode, you need two separate servers with separate ports, a reverse proxy to unify them, and CORS configuration to allow the frontend to call the API.

WebServer mode eliminates this:

```
Port 16395
├── /api/v1/*  → workflow DAG (kdeps handles)
├── /app/*     → subprocess proxy (kdeps forwards to your frontend dev server)
└── /*         → static files from publicPath
```

One port, one deployment unit. The frontend and the API ship together.

## Basic Configuration

```yaml
# workflow.yaml
settings:
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
  
  webServer:
    # Option 1: Static files only
    publicPath: "./public"    # serve from this directory
    
    # Option 2: Proxy to a running process
    command: "npm run dev"    # start this command
    appPort: 3000             # process listens on this port
    appPathPrefix: "/app"     # requests to /app/* are proxied to the process
```

## Static File Serving

```yaml
webServer:
  publicPath: "./public"
```

kdeps serves files from `publicPath` for any request that does not match an API route. Place your built frontend assets there:

```
my-agent/
├── workflow.yaml
├── resources/
└── public/
    ├── index.html
    ├── app.js
    └── style.css
```

Requests to `/`, `/index.html`, `/app.js` serve the files from `public/`. Requests to `/api/v1/chat` go to the workflow pipeline.

This is the right setup for production: build your React/Vue/Svelte app, put the build output in `public/`, package everything together with `kdeps bundle package`.

## Subprocess Proxy Mode

For development, run your frontend dev server as a subprocess:

```yaml
webServer:
  command: "npm run dev"    # command to start the frontend dev server
  appPort: 3000             # the frontend server listens here
  appPathPrefix: "/app"     # proxy requests with this prefix
  workDir: "./frontend"     # working directory for the command
```

When kdeps starts:
1. It starts `npm run dev` in `./frontend` as a child process
2. It waits for the process to be ready on port 3000
3. Requests to `/app/*` are proxied to `http://localhost:3000`

Stopping kdeps stops the subprocess too. The frontend dev server's hot-reload works through the proxy.

### React Example

```yaml
webServer:
  command: "npm run dev -- --port 3000"
  appPort: 3000
  appPathPrefix: "/app"
  workDir: "./frontend"
```

```bash
# frontend/src/api.js
const API_BASE = '/api/v1'    # relative path — works because same-origin

async function chat(message) {
  const response = await fetch(`${API_BASE}/chat`, {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({q: message})
  })
  return response.json()
}
```

No CORS configuration needed — frontend and API are on the same origin.

### Streamlit Example

```yaml
webServer:
  command: "streamlit run app.py --server.port 8501 --server.headless true"
  appPort: 8501
  appPathPrefix: "/dashboard"
  workDir: "./dashboard"
```

```python
# dashboard/app.py
import streamlit as st
import requests

st.title("My AI Agent Dashboard")

question = st.text_input("Ask something:")
if question:
    response = requests.post("/api/v1/chat", json={"q": question})
    st.write(response.json()["response"]["answer"])
```

### Gradio Example

```yaml
webServer:
  command: "python3 gradio_app.py"
  appPort: 7860
  appPathPrefix: "/ui"
```

```python
# gradio_app.py
import gradio as gr
import requests

def chat(message):
    response = requests.post("http://localhost:16395/api/v1/chat",
                             json={"q": message})
    return response.json().get("response", {}).get("answer", "Error")

iface = gr.Interface(fn=chat, inputs="text", outputs="text")
iface.launch(server_port=7860)
```

## The Request Routing Logic

```
Incoming request → path prefix check
  
/api/v1/* → apiServer → workflow DAG → JSON response
/app/*    → webServer subprocess → proxied response
/*        → webServer publicPath → static file or 404
```

You control the prefix boundaries:
- `apiServer.routes` defines the API paths
- `webServer.appPathPrefix` defines the proxied app prefix
- Everything else serves static files from `publicPath`

## Production: Building the Frontend In

For production deployments, build the frontend into the package:

```bash
# Build the React app
$ cd frontend && npm run build && cd ..

# Copy build output to public/
$ cp -r frontend/dist/* public/

# Package everything
$ kdeps bundle package workflow.yaml
```

The `.kdeps` archive includes `public/` with the built frontend. Deploy as a Docker image or binary — the frontend is part of the deployment unit. No separate frontend CDN needed.

```yaml
# Production workflow.yaml
webServer:
  publicPath: "./public"     # no subprocess needed in production
# No command: or appPort: needed
```

## WebSockets

WebSocket connections are proxied through to the subprocess:

```yaml
webServer:
  command: "node websocket-server.js"
  appPort: 3001
  appPathPrefix: "/ws"
```

```javascript
// Client
const ws = new WebSocket('ws://localhost:16395/ws/stream')
ws.onmessage = (event) => console.log(event.data)
```

WebSocket upgrade requests to `/ws/*` are proxied to `localhost:3001`. This supports streaming responses, real-time dashboards, and interactive UIs.

## Development Workflow

The recommended development setup:

```bash
# Terminal 1: run the agent with frontend dev server
$ kdeps run workflow.yaml --dev

# The agent starts on port 16395
# npm run dev starts on port 3000 (internal)
# Frontend accessible at http://localhost:16395/app
# API accessible at http://localhost:16395/api/v1/chat

# Save a resource file → kdeps reloads the workflow
# Save a frontend file → Vite/CRA hot-reloads the UI
```

`--dev` enables hot-reload for both the workflow and the subprocess. Changes to `resources/*.yaml` reload the workflow. Changes to frontend files reload through the subprocess's own HMR.

## Example: Full-Stack Agent with Dashboard

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: fullstack-agent
  version: "1.0.0"
  targetActionId: respond

settings:
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    routes:
      - path: /api/v1/analyze
        methods: [POST]
      - path: /api/v1/history
        methods: [GET]
  
  webServer:
    publicPath: "./public"         # production: serve built assets
    # command: "npm run dev"       # development: proxy to dev server
    # appPort: 5173
    # appPathPrefix: "/app"
```

```yaml
# resources/analyze.yaml
actionId: analyze
validations:
  routes: [/api/v1/analyze]
  methods: [POST]
  check:
    - get('text') != ''
chat:
  model: llama3.2:1b
  prompt: "Analyze the sentiment and key themes in: &#123;&#123; get('text') &#125;&#125;"
  jsonResponse: true
```

```yaml
# resources/history.yaml
actionId: history
validations:
  routes: [/api/v1/history]
  methods: [GET]
sql:
  connectionName: main
  query: "SELECT * FROM analyses ORDER BY created_at DESC LIMIT 20"
```

```yaml
# resources/respond.yaml
actionId: respond
requires: [analyze, history]
apiResponse:
  success: true
  response:
    analysis: get('analyze')
    recent: get('history')
```

The frontend in `public/` calls `/api/v1/analyze` and `/api/v1/history` directly. The whole application ships as one Docker image.

X> ## Exercise
X>
X> Build a minimal AI-powered web app that serves a static HTML frontend alongside the chatbot API — all from a single kdeps server on one port.
X>
X> 1. Configure `webServer.publicPath: "./public"` in `workflow.yaml`.
X> 2. Create `public/index.html` with a minimal chat interface: a text input, a submit button, and a `<div>` that shows the response. Use plain JavaScript `fetch()` to call `POST /api/v1/chat`.
X> 3. Start kdeps: `kdeps run workflow.yaml`. Open `http://localhost:16395` in a browser and verify the HTML page loads.
X> 4. Type a question in the input and submit. Verify the LLM response appears in the `<div>` without a page reload.
X> 5. Check the browser's Network tab. Confirm the HTML came from port `16395` and the API call also went to port `16395` — there is no second server and no CORS header needed.
X>
X> **Stretch goal:** Add a second route `/history` that serves a different static HTML page showing a table of all past questions and answers (stored in session). Configure `webServer` to proxy `/api/v1/stream` to a local development server on port 3000 running a React frontend, and verify kdeps forwards the request correctly.
