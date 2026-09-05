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
- Opt into the standard "send this page" bookmarklet with `KDEPS_WASM_CAPTURE`
- Handle a captured page via the `kdeps:capture` event

The backend, model, and API key come from the [settings drawer](/deployment/wasm#settings-drawer) kdeps injects into every `--wasm` app - you do not build a key field.

## Before you start

- kdeps installed (`kdeps --version`). `--wasm` compiles `kdeps.wasm` itself.
- An API key for any cloud backend (OpenAI, Anthropic, Groq, ...)

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
  agentSettings:
    timezone: UTC
    env:
      KDEPS_DEFAULT_BACKEND: openai        # default; the drawer can switch it
      KDEPS_WASM_CAPTURE: "url,title,text"  # add the "send this page" bookmarklet
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

## Step 4: the result page

Put `data/public/index.html` next to the workflow; the bundler copies `data/` into `dist/`. The settings drawer and the bookmarklet are added by kdeps - your page only needs to render the answer and handle a capture:

```html
<!-- data/public/index.html (trimmed) -->
<div id="kdeps-capture-cta"></div>   <!-- bundler drops the bookmarklet link here -->
<textarea id="text" placeholder="Or paste text and click Summarize."></textarea>
<button id="run">Summarize</button>
<pre id="out"></pre>
<script>
  async function summarize(input) {
    if (!window.__kdepsSettingsReady()) { /* prompt user to open the gear menu */ return; }
    document.getElementById('out').textContent =
      (await window.kdeps.execute(input)).response.summary;
  }
  document.getElementById('run').onclick = function () {
    summarize({ text: document.getElementById('text').value });
  };
  // The bookmarklet delivers the captured page here.
  window.addEventListener('kdeps:capture', function (e) { summarize(e.detail); });
</script>
```

The full file is in [`examples/page-summarizer/data/public/index.html`](https://github.com/kdeps/kdeps/blob/main/examples/page-summarizer/data/public/index.html).

## Step 5: build and double-click

```bash
kdeps bundle build . --wasm
# default --wasm-output html
```

That writes `page-summarizer.html` next to the workflow. Double-click it. No Docker, no `http.server`. Use `--wasm-output server` if you want `{name}-wasm/` plus an nginx image.

1. Click the gear (top-right), pick a backend + model, paste an API key
2. Drag **Send this page to page-summarizer** onto the bookmarks bar
3. Open any article, click the bookmark - the summarizer opens and runs

Paste text and click Summarize if you do not want a bookmarklet.

## Summary

You built a WASM agent that:

- Allowlists only `chat:` + `apiResponse:` (WASM cannot scrape or run local models)
- Opts into the standard bookmarklet with `KDEPS_WASM_CAPTURE`, then reads the captured page from the `kdeps:capture` event
- Gets its backend, model, and key from the kdeps settings drawer, not a hand-built form

## See also

- [WASM web app](/deployment/wasm) - allowlist, `file://`, cloud `chat:`
- [Chat web app](/examples/chat-web-app) - API + static UI without WASM
- [LLM resource](/resources/llm/) - `scenario:`, models, backends
