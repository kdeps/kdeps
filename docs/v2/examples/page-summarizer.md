# Summarize the current page (WASM bookmarklet)

A bookmarklet copies the current tab's text into a kdeps WASM app. The app runs a cloud `chat:` in the browser and shows a summary. No kdeps install for the reader.

*Applies to workflow mode.*

```d2
direction: right
Tab: current page {shape: rectangle}
BM: bookmarklet {shape: rectangle}
App: page-summarizer.html {shape: rectangle}
LLM: OpenAI {shape: rectangle}
Tab -> BM: innerText
BM -> App: postMessage
App -> LLM: chat
LLM -> App: summary
```

WASM cannot read the open tab. The bookmarklet runs *in* that tab and sends `url`, `title`, and `text`.

This tutorial is for developers who have completed the [quickstart](/getting-started/quickstart) and [WASM web app](/deployment/wasm). It assumes you know:

- Basic YAML
- How to drag a bookmarklet onto the bookmarks bar

By the end you will be able to:

- Build a WASM-only workflow (`chat` + `apiResponse`)
- Ship a bookmarklet that feeds the current page into `window.kdeps.execute`
- Pass an API key at runtime with `window.kdeps.init`

## Before you start

- kdeps installed, plus `make build-wasm` so `kdeps.wasm` sits next to the CLI
- An OpenAI API key

## Step 1: create the project

```bash
mkdir page-summarizer
cd page-summarizer
mkdir -p resources data/public
```

## Step 2: workflow

WASM rejects local models. Set a cloud backend and a hosted model.

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: page-summarizer
  version: "1.0.0"
  description: "Summarize the current browser page. WASM + bookmarklet."
  targetActionId: response

settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/summarize
        methods: [POST]
        public: true  # bookmarklet cannot send a bearer token
  webServer:
    routes:
      - path: /
        serverType: static
        publicPath: ./data/public
  agentSettings:
    timezone: UTC
    env:
      KDEPS_DEFAULT_BACKEND: openai  # WASM rejects ollama / llamafile / gguf
```

```yaml
# kdeps.pkg.yaml
name: page-summarizer
version: "1.0.0"
type: workflow
description: "WASM bookmarklet that summarizes the current browser page"
license: Apache-2.0
tags: [wasm, bookmarklet, chat]
```

## Step 3: resources

<div v-pre>

```yaml
# resources/summarize.yaml
actionId: summarize
name: Summarize page
validations:
  methods: [POST]
  routes: [/api/v1/summarize]
  check:
    - get('text') != ''
  error:
    code: 400
    message: "'text' is required"
chat:
  model: gpt-4o-mini  # hosted name; llama3.2:1b is rejected in WASM
  role: user
  timeout: 60s
  scenario:
    - role: system
      prompt: Summarize for a busy reader. 5-8 sentences, then 3-5 bullets.
  prompt: |
    URL: {{ get('url') }}
    Title: {{ get('title') }}

    Page text:
    {{ get('text') }}
```

```yaml
# resources/response.yaml
actionId: response
name: API Response
requires: [summarize]
apiResponse:
  success: true
  response:
    url: "{{ get('url') }}"
    title: "{{ get('title') }}"
    summary: "{{ get('summarize').message.content }}"
```

</div>

## Step 4: the page and bookmarklet

Put `data/public/index.html` next to the workflow. The WASM bundler copies `data/` into `dist/`. The page:

1. Listens for `kdeps:ready`
2. Stores the API key in `localStorage`
3. Calls `window.kdeps.init({ OPENAI_API_KEY, KDEPS_DEFAULT_BACKEND: "openai" })`
4. Calls `window.kdeps.execute({ url, title, text })`
5. Exposes a `javascript:` bookmarklet. Drag that control (this HTML page) onto the bookmarks bar. Clicking it on any tab copies `innerText` and opens this same file. No web server.

The full file is in [`examples/page-summarizer/data/public/index.html`](https://github.com/kdeps/kdeps/blob/main/examples/page-summarizer/data/public/index.html). Copy it.

## Step 5: build and double-click

```bash
make build-wasm
kdeps bundle build . --wasm
```

That writes `page-summarizer.html` next to the workflow. Double-click it. No Docker, no `http.server`.

1. Paste the OpenAI API key
2. Drag **Summarize page** onto the bookmarks bar
3. Open any article, click the bookmark

If the browser blocks `window.open` of a `file://` URL, the bookmark copies the page to the clipboard - focus the HTML window and it reads it.

Paste text and click Summarize if you do not want a bookmarklet.

## Summary

You built a WASM agent that:

- Allowlists only `chat:` + `apiResponse:` (WASM cannot scrape or run local models)
- Takes page text from a bookmarklet, not from the WASM runtime
- Injects the API key at runtime with `window.kdeps.init`

## See also

- [WASM web app](/deployment/wasm) - allowlist, `file://`, cloud `chat:`
- [Chat web app](/examples/chat-web-app) - API + static UI without WASM
- [LLM resource](/resources/llm/) - `scenario:`, models, backends
