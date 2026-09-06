# WASM web app

`kdeps bundle build --wasm` compiles `kdeps.wasm` and writes a browser app.

*Applies to workflow mode.*

## Build

```bash
# both: a standalone HTML file AND a served site (no Docker needed)
kdeps bundle build examples/page-summarizer --wasm

# just one
kdeps bundle build examples/page-summarizer --wasm=standalone
kdeps bundle build examples/page-summarizer --wasm=server

# a Docker image too - only when you pass --tag
kdeps bundle build examples/page-summarizer --wasm=server --tag myorg/summ:1
```

| `--wasm` | What you get |
| --- | --- |
| _(bare)_ | both of the below |
| `standalone` | `{name}.html` - one file with everything inlined; double-click it |
| `server` | `{name}-wasm/` - `index.html` + `kdeps.wasm` + JS, plus `package.json` (`npm run server`), `serve.sh` (no Node), `serve.json`, and `nginx.conf` / `Dockerfile` |
| `none` | nothing (same as omitting `--wasm`) |

Serve the site locally with `cd {name}-wasm && npm run server` (or `./serve.sh`), then open `http://localhost:3000`. Both scripts set the `application/wasm` MIME type and `Access-Control-Allow-Origin: *` so the bookmarklet can load the module cross-origin.

`--wasm` compiles `kdeps.wasm` (`go build GOOS=js GOARCH=wasm`) if it is not already next to the CLI or in `KDEPS_WASM_BINARY`. Needs Go and the kdeps source tree (or a release that ships `kdeps.wasm`).

`standalone` inlines `wasm_exec.js`, the WASM binary, the settings drawer, and bootstrap so `file://` does not CORS on open. `server` loads `kdeps.wasm` with `fetch`, so it needs HTTP.

The runtime compile (~1-2s for the embedded module) is deferred until after the page has painted, and the module is decoded via a `data:` URL (native, off the JS thread) rather than a multi-megabyte synchronous loop. It still compiles on the main thread - a Web Worker can't be used from `file://` - so `standalone` builds cover the page with a **blocking spinner overlay** until the runtime is ready, then reveal the app. (A page that ships its own `#kdeps-loading` element keeps that instead.) `server` builds skip the overlay. Listen for `kdeps:loading`, `kdeps:ready`, and `kdeps:error` on `window` to drive your own UI.

Bookmarklet sample: [`examples/page-summarizer`](https://github.com/kdeps/kdeps/tree/main/examples/page-summarizer) for both builds.

Init and `build --wasm` reject any resource the WASM runtime cannot execute.

## Settings drawer

Every `--wasm` app ships a settings drawer (a gear button, top-right). The viewer picks the cloud backend, a model, and pastes an API key; kdeps stores the choice and re-runs `kdepsInit` with the matching env (`KDEPS_DEFAULT_BACKEND`, `KDEPS_WASM_MODEL`, `<PROVIDER>_API_KEY`) whenever it changes. No key is baked into the file.

- Backend dropdown is the full cloud provider list; the model box is an editable combo pre-filled from the model catalog for the chosen backend. It defaults to the workflow's `chat.model`, or the backend's default model when the resource uses `model: system`.
- The drawer is "the system" for a WASM app: its backend and model apply to **every** chat resource at runtime, even ones that hardcode a model the chosen backend cannot serve. Write `backend: system` / `model: system` to make that intent explicit.
- To render the panel inside your own layout instead of the floating drawer, put `<div id="kdeps-settings"></div>` in your `index.html`.
- `window.__kdepsSettingsReady()` returns `true` once a backend and (if needed) a key are set - gate your submit button on it.

### Persistence, sharing, and import

- **Shared across apps.** The choice is saved to one browser-wide store (`localStorage` key `kdeps.settings`), so an API key set in any kdeps WASM app is reused by all of them on that browser. A per-app store (`kdeps.<metadata.name>.settings`) still wins when present.
- **Export / Import** buttons in the drawer download and load a `kdeps-settings.json` file - move your setup between browsers or machines.
- **Import machine settings.** `kdeps bundle build --wasm` bakes the build machine's own `~/.kdeps/config.yaml` LLM defaults (backend, first model, base URL - **never cloud API keys**) into the app. One drawer click adopts them.
- **`--wasm-embed-secrets`** additionally bakes this machine's real credentials into that same "Import machine settings" data: every cloud API key from `~/.kdeps/config.yaml` (`*_api_key`), and the m365 auth from `~/.config/kdeps/m365/token-cache.json` + `secrets.json`. Importing loads the keys per backend; m365 auth is replayed as `M365_TOKEN_CACHE_JSON` / `M365_SECRETS_JSON` when the m365 backend is selected. Off by default: **the build then contains real credentials - do not commit or share it.** A build-time warning is printed.
- **`m365` backend.** Selecting `m365` (or any local OpenAI-compatible proxy) swaps the API-key field for a **Base URL** field. The env becomes `KDEPS_DEFAULT_BACKEND=openai` + `KDEPS_LLM_BASE_URL=<url>`. Run `kdeps` locally so the proxy is up; point the field at its address.

So a WASM resource can just defer everything to the drawer:

```yaml
settings:
  agentSettings:
    env:
      KDEPS_DEFAULT_BACKEND: openai   # first-load default; the drawer changes it
chat:
  backend: system                    # follow the drawer
  model: system                      # follow the drawer
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

The bookmarklet runs in whatever tab it is clicked in, reads `location.href`, `document.title`, and `document.body.innerText` (capped at 24k chars), and maps them onto the listed field names.

### The widget

Clicking the bookmarklet shows a small kdeps panel that:

1. shows what the run is doing (reading the page -> loading -> running -> done);
2. renders an input box **only if the workflow needs a field the capture does not provide** (kdeps works this out from `validations`);
3. renders the result as markdown, and grows to fit;
4. never closes on its own - only its **x** / Dismiss button (or a re-run) closes it. The **gear** in the header opens the settings drawer.

**Where it appears depends on how the app is served:**

| App | Bookmarklet behaviour |
|---|---|
| served app (`--wasm=server` or `--wasm`) | Injects `kdeps-bookmarklet.js` from the app's origin **into the page you're reading** - a floating panel, **no popup window**. Needs the origin to send CORS on `.wasm`/`.js` (the generated `nginx.conf` does). If a strict site CSP blocks the inject, it falls back to opening the app in a tab. |
| `file://` standalone (`--wasm=standalone` or `--wasm`) | Opens the app itself as a **small popup window** (`#kdeps-widget`) - a bookmarklet can't carry the ~45 MB module and `file://` scripts are cross-origin-blocked from a web page, so a one-time "allow popups" may be needed. |

The bookmarklet link is injected into `<div id="kdeps-capture-cta"></div>` if your HTML has one, and always into the settings drawer. Without `KDEPS_WASM_CAPTURE` there is no bookmarklet. Your app's own `index.html` UI is used only when the file/URL is opened directly, not from the bookmarklet.

## Allowed resources

| YAML key | Why it works |
| --- | --- |
| `chat:` | HTTP calls to hosted providers (openai, anthropic, groq, xai, google, ...) |
| `httpClient:` | Browser `fetch` |
| `apiResponse:` | Formats the HTTP result; no I/O |

Expressions, `before:` / `after:` (bare expr only), `items:`, and `loop:` are fine.

Set an online backend **and** a hosted model, **or** use `backend: system` / `model: system` and let the [settings drawer](#settings-drawer) supply them at runtime (this is the usual choice for a WASM app). An empty `model:`, Ollama tags (`llama3.2:1b`), `.gguf`, `.llamafile`, `router`, and `auto-router` all error at init - `system` does not.

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
