# Chatbot Example

Simple LLM chatbot using the unified API.

## Features

- ✅ YAML configuration
- ✅ Unified API (`get()` function)
- ✅ LLM chat with Ollama
- ✅ JSON response
- ✅ Validation with preflight checks

## Run Locally

```bash
# From examples/chatbot directory
kdeps run workflow.yaml

# Or from root
kdeps run examples/chatbot/workflow.yaml
```

## Test

```bash
curl -X POST http://localhost:3000/api/v1/chat \
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
