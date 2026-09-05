# WASM web app

`kdeps bundle build --wasm` ships the workflow as a static site. The browser loads `kdeps.wasm` and runs the DAG there. No local LLM, no shell, no database sockets.

*Applies to workflow mode.*

## Build

```bash
make build-wasm
kdeps bundle build examples/http-advanced --wasm
# Docker image, or copy the generated dist/ folder
docker run -p 80:80 kdeps-wasm:latest
```

Or open `dist/index.html` from Finder/Explorer (no kdeps, no server).

`kdeps.wasm` must sit next to the CLI, or set `KDEPS_WASM_BINARY` and `KDEPS_WASM_EXEC_JS`. GitHub releases attach both files.

The `dist/` folder is a self-contained site. Double-click `dist/index.html` - `wasm_exec.js`, the WASM binary, and the bootstrap are inlined into that one file. The page never `fetch()`es local assets, so `file://` does not CORS on open. You can copy just `index.html`.

Sibling files (`kdeps.wasm`, `wasm_exec.js`, `kdeps-wasm-embed.js`, `kdeps-bootstrap.js`) stay in `dist/` for HTTP/Docker. nginx uses them; the HTML does not.

Cloud `chat:` from `file://` still needs the provider to allow origin `null`. If the model call fails, serve the folder over HTTP (`python3 -m http.server 8080`) or use the Docker image.

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
