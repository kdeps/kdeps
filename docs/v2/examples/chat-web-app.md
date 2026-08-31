# Build a chat web app

*Applies to workflow mode.*

## Overview

In this tutorial you build a single kdeps project that serves both a chat API
and the static web page that talks to it. The browser posts messages to
`/api/v1/chat`; kdeps runs an LLM and returns the reply.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- Basic HTML and JavaScript (`fetch`)

By the end you will be able to:

- Serve an API and a static frontend from one workflow
- Mark a route `public` so a browser can call it without a bearer token
- Set a system prompt with `scenario:`
- Return a friendly message on LLM failure with `onError`

## Background

kdeps runs an `apiServer:` and a `webServer:` at the same time. The API server
handles JSON routes; the web server serves files from a directory. A browser
frontend has no safe place to keep an auth token, so its API route is marked
`public: true` and protected by CORS instead.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.

## Step 1: create the project

```bash
mkdir chat-app
cd chat-app
mkdir resources public
```

## Step 2: configure both servers

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: chat-app
  version: "1.0.0"
  targetActionId: chat

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
        public: true              # browser calls this without a token
    cors:
      allowOrigins: ["*"]
      allowMethods: [GET, POST, OPTIONS]
      allowHeaders: [Content-Type]
  webServer:
    routes:
      - path: /
        serverType: static
        publicPath: ./public      # the frontend
```

## Step 3: the chat resource

Create `resources/chat.yaml`:

<div v-pre>

```yaml
# resources/chat.yaml
actionId: chat
name: Chat handler
validations:
  methods: [POST]
  routes: [/api/v1/chat]
  required:
    - message                     # 400 if missing
  rules:
    - field: message
      type: string
      minLength: 1
      message: "message cannot be empty"
chat:
  model: llama3.2:1b
  role: user
  prompt: "{{ get('message') }}"
  scenario:
    - role: system
      prompt: "You are a helpful assistant. Answer clearly and concisely."
  timeout: 300s
onError:
  action: continue
  fallback:
    reply: "Sorry, something went wrong. Please try again."
apiResponse:
  success: true
  response:
    reply: "{{ get('chat').message.content }}"
```

</div>

`scenario:` prepends messages to the conversation - here a single `system`
message. `onError: continue` with a `fallback` means an LLM timeout returns a
polite message instead of a 500.

## Step 4: the frontend

Create `public/index.html`:

```html
<!doctype html>
<html>
<head><meta charset="utf-8"><title>Chat</title></head>
<body>
  <div id="log"></div>
  <form id="f">
    <input id="msg" autocomplete="off" placeholder="Say something" />
    <button>Send</button>
  </form>
  <script>
    const log = document.getElementById('log');
    document.getElementById('f').onsubmit = async (e) => {
      e.preventDefault();
      const message = document.getElementById('msg').value;
      log.innerHTML += `<p><b>You:</b> ${message}</p>`;
      const r = await fetch('/api/v1/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message }),
      });
      const data = await r.json();
      log.innerHTML += `<p><b>Bot:</b> ${data.data.reply}</p>`;
      document.getElementById('msg').value = '';
    };
  </script>
</body>
</html>
```

## Step 5: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token   # still required to start the server
kdeps run .
```

Open `http://localhost:16395/` and send a message. Or test the API directly -
no `Authorization` header, because the route is public:

```bash
curl -X POST http://localhost:16395/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Say hi in one word."}'
```

## Summary

You built a project that:

- Serves an API (`apiServer:`) and a frontend (`webServer:`) together
- Exposes the chat route as `public` so the browser needs no token
- Sets a system prompt with `scenario:`
- Degrades gracefully on LLM failure with `onError`

## Next steps

- [Static site tutorial](/examples/static-site) - web server mode on its own
- [LLM resource](/resources/llm) - `scenario:`, streaming, JSON mode
- [CORS configuration](/configuration/cors) - locking down origins
- [Error handling (onError)](/concepts/error-handling) - retry, fallback, `when`
