# WASM web app

`kdeps bundle build --wasm` compiles `kdeps.wasm` and writes a single HTML file. Double-click it. No web server, no Docker, no nginx.

*Applies to workflow mode.*

## Build

```bash
kdeps bundle build examples/page-summarizer --wasm
# writes examples/page-summarizer/page-summarizer.html
```

`--wasm` compiles `kdeps.wasm` (`go build GOOS=js GOARCH=wasm`) if it is not already next to the CLI or in `KDEPS_WASM_BINARY`. Needs Go and the kdeps source tree (or a release that ships `kdeps.wasm`).

The HTML is self-contained: `wasm_exec.js`, the WASM binary, and the bootstrap are inlined, so `file://` does not CORS on open.

Bookmarklet sample: [`examples/page-summarizer`](https://github.com/kdeps/kdeps/tree/main/examples/page-summarizer). Drag the HTML onto the bookmarks bar. The bookmarklet sends `innerText`; WASM cannot read the open page itself.

Init and `build --wasm` reject any resource the WASM runtime cannot execute.

## Allowed resources

| YAML key | Why it works |
| --- | --- |
| `chat:` | HTTP calls to hosted providers (openai, anthropic, groq, xai, google, ...) |
| `httpClient:` | Browser `fetch` |
| `apiResponse:` | Formats the HTTP result; no I/O |

Expressions, `before:` / `after:` (bare expr only), `items:`, and `loop:` are fine.

Set an online backend **and** a hosted model. Empty `KDEPS_DEFAULT_BACKEND` defaults to `file` (llamafile). Empty `model:` defaults to `llama3.2:1b`. Ollama tags (`llama3.2:1b`), `.gguf`, `.llamafile`, `router`, and `auto-router` error at init.

```yaml
settings:
  agentSettings:
    env:
      KDEPS_DEFAULT_BACKEND: openai
# ...
chat:
  model: gpt-4o
  prompt: "{{ get('q') }}"
```

Pass the API key at init (`kdepsInit(yaml, { OPENAI_API_KEY: "..." })`) or bake it in `env:` (it will be visible in the page).

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
