# LLM (`chat:`)

Workflow (and agency) LLM calls use the **`chat:`** action. Agent mode uses the same backends for its loop.

<div v-pre>

```yaml
actionId: llm
chat:
  model: llama3.2:1b
  role: user
  prompt: "{{ get('q') }}"
  timeout: 60s
  systemPrompt: |
    Be concise. Prefer bullet answers.
  temperature: 0.7
  maxTokens: 1000
  jsonResponse: false
```

</div>

Output: `get('llm')` — reply text usually at `.message.content` (shape can vary by backend).

## Useful fields

| Field | Role |
|-------|------|
| `model` | Model id, or routing name from config |
| `prompt` / `systemPrompt` | User / system text |
| `scenario` / `messages` | Multi-turn seed / history |
| `jsonResponse` | Ask for JSON and parse |
| `files` | Vision attachments |
| `componentTools` | Expose installed components as tools |
| `timeout` | Hard wait limit |
| Sampling | `temperature`, `topP`, `maxTokens`, … |

Structured extract:

```yaml
chat:
  jsonResponse: true
  prompt: |
    Extract JSON: company_name, founded_year
    Text: {{ get('body') }}
```

## Backends (machine config)

Set in `~/.kdeps/config.yaml` or env — not usually in each resource.

| Backend | Notes |
|---------|--------|
| `file` (llamafile) | Default local; self-serving binary |
| `gguf` | llama-server / GGUF |
| `ollama` | Local Ollama |
| `openai` | OpenAI or any `/v1` compat (`base_url`) |
| `anthropic`, `groq`, `google`, `deepseek`, `xai`, … | Provider keys |

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1
  # openai_api_key: "..."
```

```bash
export KDEPS_DEFAULT_BACKEND=openai
export KDEPS_LLM_BASE_URL=http://host:8000/v1
export OPENAI_API_KEY=...
export ANTHROPIC_API_KEY=...
```

Appliance path: [LLM server](/llm-server). Agent loop flags: [Coding agent](/agent) · [Config](/config).
