# WASM web app

`kdeps bundle build --wasm` compiles `kdeps.wasm` and writes a browser app. `--wasm-output` picks the shape.

*Applies to workflow mode.*

## Build

```bash
# html (default): one file, double-click, no server
kdeps bundle build examples/page-summarizer --wasm
kdeps bundle build examples/page-summarizer --wasm --wasm-output html

# server: static site + nginx image (needs HTTP)
kdeps bundle build examples/page-summarizer --wasm --wasm-output server
docker run -p 80:80 kdeps-wasm:latest
```

| `--wasm-output` | What you get |
| --- | --- |
| `html` (default) | `{name}.html` with WASM inlined. Double-click it. |
| `server` | `{name}-wasm/` (`index.html` + `kdeps.wasm` + JS) and an nginx Docker image |

`--wasm` compiles `kdeps.wasm` (`go build GOOS=js GOARCH=wasm`) if it is not already next to the CLI or in `KDEPS_WASM_BINARY`. Needs Go and the kdeps source tree (or a release that ships `kdeps.wasm`).

`html` inlines `wasm_exec.js`, the WASM binary, the settings drawer, and bootstrap so `file://` does not CORS on open. `server` loads `kdeps.wasm` with `fetch`, so it needs HTTP.

Bookmarklet sample: [`examples/page-summarizer`](https://github.com/kdeps/kdeps/tree/main/examples/page-summarizer) (`html` output).

Init and `build --wasm` reject any resource the WASM runtime cannot execute.

## Settings drawer

Every `--wasm` app ships a settings drawer (a gear button, top-right). The viewer picks the cloud backend, a model, and pastes an API key; kdeps stores the choice in that browser's `localStorage` (key `kdeps.<metadata.name>.settings`) and re-runs `kdepsInit` with the matching env (`KDEPS_DEFAULT_BACKEND`, `KDEPS_WASM_MODEL`, `<PROVIDER>_API_KEY`) whenever it changes. No key is baked into the file.

- Backend dropdown is the full cloud provider list; the model box is an editable combo pre-filled from the model catalog for the chosen backend, defaulting to the workflow's `chat.model`.
- `KDEPS_WASM_MODEL` overrides `chat.model` on every chat resource at runtime, so switching backend + model in the drawer just works even though the YAML names one model.
- To render the panel inside your own layout instead of the floating drawer, put `<div id="kdeps-settings"></div>` in your `index.html`.
- `window.__kdepsSettingsReady()` returns `true` once a backend and (if needed) a key are set - gate your submit button on it.

So `env:` only needs the backend (and only as a default the drawer can change):

```yaml
settings:
  agentSettings:
    env:
      KDEPS_DEFAULT_BACKEND: openai   # default; the drawer can switch it
chat:
  model: gpt-4o                       # default; KDEPS_WASM_MODEL overrides it
  prompt: "{{ get('q') }}"
```

Baking a key into `env:` still works but it is then visible in the page - prefer the drawer.

## Page-capture bookmarklet

Set `KDEPS_WASM_CAPTURE` to a comma list of workflow input fields and the bundle adds a draggable "Send this page to \<name\>" bookmarklet plus a receiver:

```yaml
settings:
  agentSettings:
    env:
      KDEPS_WASM_CAPTURE: "url,title,text"
```

The bookmarklet runs in whatever tab it is clicked in, reads `location.href`, `document.title`, and `document.body.innerText` (capped at 24k chars), maps them onto the listed field names, and hands them to the app via `postMessage` / clipboard / `window.name`. The app receives a `kdeps:capture` CustomEvent whose `detail` is the field map:

```js
window.addEventListener('kdeps:capture', function (e) {
    window.kdeps.execute(e.detail);   // { url, title, text }
});
```

The bookmarklet link is injected into `<div id="kdeps-capture-cta"></div>` if your HTML has one, and always into the settings drawer. Without `KDEPS_WASM_CAPTURE` there is no bookmarklet.

## Allowed resources

| YAML key | Why it works |
| --- | --- |
| `chat:` | HTTP calls to hosted providers (openai, anthropic, groq, xai, google, ...) |
| `httpClient:` | Browser `fetch` |
| `apiResponse:` | Formats the HTTP result; no I/O |

Expressions, `before:` / `after:` (bare expr only), `items:`, and `loop:` are fine.

Set an online backend **and** a hosted model. Empty `KDEPS_DEFAULT_BACKEND` defaults to `file` (llamafile). Empty `model:` defaults to `llama3.2:1b`. Ollama tags (`llama3.2:1b`), `.gguf`, `.llamafile`, `router`, and `auto-router` error at init. The [settings drawer](#settings-drawer) supplies the backend, model, and key at runtime.

## Rejected resources

Anything else errors at init / `build --wasm`: `sql`, `exec`, `python`, `scraper`, `embedding`, `searchLocal`, `searchWeb`, `browser`, `file`, `git`, `email`, `botReply`, `telephony`, `ocr`, `transcribe`, `vectorStore`, `loader`, `codeIntelligence`, `agent`, `component`.

`sql:` is compiled with Postgres/MySQL drivers but the browser cannot open a TCP socket, so it is not allowed.

## Tools

`chat.tools` may only call another **chat** or **httpClient** resource via `script:`.

```yaml
chat:
  prompt: "{{ get('q') }}"
  tools:
    - name: lookup
      description: Fetch a URL
      script: fetchURL   # actionId of an httpClient resource
      parameters:
        url:
          type: string
          required: true
```

These tool forms error:

| Tool | Why |
| --- | --- |
| `mcp:` (stdio or SSE) | WASM cannot spawn a process or dial MCP over TCP |
| `componentTools:` | Components are not in the WASM registry |
| `script:` pointing at `sql` / `exec` / `python` / ... | Same allowlist as resources |
| `script:` that is not an `actionId` in this workflow | Missing target |
| `chat.files` local paths | WASM cannot read the disk; use `https://` URLs |

Agent-loop built-in tools (`bash_exec`, `sql_*`, `web_search`, ...) are not part of WASM workflow mode.
