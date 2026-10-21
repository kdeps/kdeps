# Action tools

`http_request`, `search_local`, `transcribe_audio`, `transcribe_audio_offline`, `ocr_image`, and `load_document` are always on. Zapier and Google cache register when their credentials are set (the Zapier tools are [declared HTTP tools](/agent/tools#declared-http-tools)). Commits the agent creates carry a co-author trailer. Permission modes stay on [Built-in tools](/agent/tools).

## Git commit attribution

Commits the agent creates carry a co-author trailer naming kdeps and the model that wrote them. Switching models mid-session with `/model` is normal, so the trailer records which one was actually driving:

```text
Co-Authored-By: kdeps (deepseek/deepseek-reasoner) <noreply@kdeps.com>
```

How the model is named depends on where it runs:

| Model | Trailer |
|-------|---------|
| Cloud provider | `kdeps (deepseek/deepseek-reasoner) <noreply@kdeps.com>` |
| Cloud provider | `kdeps (openai/gpt-4o-mini) <noreply@kdeps.com>` |
| Ollama | `kdeps (ollama/llama3.2) <noreply@kdeps.com>` |
| llamafile | `kdeps (hfuser/gemma4-2-9b llamafile) <noreply@kdeps.com>` |
| GGUF | `kdeps (hfuser/gemma4-2-9b gguf) <noreply@kdeps.com>` |

Cloud and Ollama models are namespaced by their provider (`provider/model`). Local llamafile and GGUF models already carry their HuggingFace namespace in the name, so the runtime is appended instead - the same repo is often published as both, and the name alone cannot tell them apart.

With no model configured, the trailer falls back to `Co-Authored-By: kdeps <noreply@kdeps.com>`.

A configured [identity](/reference/advanced-config#agent-identity) takes priority over all of the above: with `identity.name`/`identity.email` set, the trailer becomes `Co-Authored-By: Sales Bot <sales-bot@example.com>` instead of naming the model.

## Actions and integrations

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `zapier_list_actions` | `ZAPIER_NLA_API_KEY` | List available Zapier NLA actions |
| `zapier_run_action` | `ZAPIER_NLA_API_KEY` | Execute a Zapier NLA action |
| `google_cache_create` | `GOOGLE_API_KEY` | Create a Google AI cached content object |
| `google_cache_list` | `GOOGLE_API_KEY` | List Google AI cached content objects |
| `google_cache_delete` | `GOOGLE_API_KEY` | Delete a Google AI cached content object |

`google_cache_create`, `google_cache_delete`, and `google_cache_list` run through the [kdeps API](/reference/api). The agent registers them for the model only when `GOOGLE_API_KEY` is set. A call with the key unset returns `google_cache_create: disabled (GOOGLE_API_KEY is unset)`, and the same sentence for `google_cache_delete` and `google_cache_list`.

## Resource-backed tools

These always-on tools invoke the corresponding kdeps executor directly:

| Tool | Description |
|------|-------------|
| `http_request` | Make an HTTP request (GET/POST/PUT/DELETE/PATCH) |
| `search_local` | Search files in a folder (`path`), or, with no `path`, every folder of the working project: its own and its [search folders](/agent/knowledge#search-folders), best hits of each in turn. Dot-directories, vendor/build folders (`node_modules`, `vendor`, `target`, caches), and files over 1 MiB are skipped. Key and credential files are never returned |
| `transcribe_audio` | Transcribe an audio file through a Whisper API (OpenAI, Groq, or a local HTTP server); a declared HTTP tool |
| `transcribe_audio_offline` | Transcribe offline with whisper.cpp (`whisper-cli`), no API key; a declared exec tool |
| `ocr_image` | Extract text from an image via tesseract (local, no API key); a [declared exec tool](/agent/tools#declared-exec-tools) |
| `load_document` | Load and extract text from a document |
