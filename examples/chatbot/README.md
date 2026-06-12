# Chatbot Example (Ollama backend)

Simple LLM chatbot that explicitly opts into the ollama backend.

kdeps defaults to the `file` backend (llamafile, no server needed - see
`examples/llamafile-chat/`). This example pins ollama instead via
`KDEPS_DEFAULT_BACKEND: ollama` in `agentSettings.env` and bakes the ollama
server into container builds with `installOllama: true`.

## Features

- ✅ YAML configuration
- ✅ Unified API (`get()` function)
- ✅ LLM chat with Ollama (explicit opt-in)
- ✅ JSON response
- ✅ Validation with preflight checks

## Run Locally

Requires the [ollama](https://ollama.com) CLI; kdeps starts the server
automatically if it is not already running.

```bash
# From examples/chatbot directory
kdeps run workflow.yaml

# Or from root
kdeps run examples/chatbot/workflow.yaml
```

## Test

```bash
curl -X POST http://localhost:16395/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"q": "What is artificial intelligence?"}'
```

## Response

```json
{
  "data": {
    "answer": "Artificial intelligence (AI) is..."
  },
  "query": "What is artificial intelligence?"
}
```

## Structure

```
chatbot/
├── workflow.yaml              # Main workflow configuration
└── resources/
    ├── llm.yaml              # LLM chat resource
    └── response.yaml         # API response resource
```

## Key Concepts

### Unified API

Uses `get()` for all data access:

```yaml
# Get query parameter
prompt: "{{ get('q') }}"

# Get LLM response from previous resource
data: get('llmResource')

# Validation
validations:
  - get('q') != ''
```

### Auto-Detection

`get()` automatically detects the data source:
- `get('q')` → Query parameter
- `get('llmResource')` → Resource output
- `get('user_data')` → Memory storage
