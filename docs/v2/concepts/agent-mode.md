# Agent Mode

Agent mode (`kdeps serve`) runs a workflow as a live LLM-driven agent. Every
resource, component, and sub-agent defined in the workflow is auto-registered as
a callable tool so the LLM can invoke them during a conversation.

## How it works

1. `kdeps serve workflow.yaml` loads the workflow and builds a tool registry.
2. All workflow resources become tools (tool name = `actionId`).
3. All components become tools (tool name = component `metadata.name`).
4. fformat built-in tools (JSON/YAML/CSV/XML) are always available.
5. The agent loop runs a synthetic chat resource with the full tool set attached.
6. The engine's existing tool-call dispatch handles every LLM function call.

The workflow engine is unchanged - agent mode is additive. `kdeps run` continues
to work exactly as before.

## Command

```bash
kdeps serve workflow.yaml [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--model` | `KDEPS_AGENT_MODEL` env or `llama3.2` | LLM model name |
| `--backend` | `KDEPS_AGENT_BACKEND` env or `ollama` | LLM backend |
| `--base-url` | `KDEPS_AGENT_BASE_URL` env | LLM API base URL |
| `--system` | (none) | System prompt injected at conversation start |
| `--debug` | false | Enable debug logging |

### Environment variables

```bash
KDEPS_AGENT_MODEL=llama3.2
KDEPS_AGENT_BACKEND=ollama
KDEPS_AGENT_BASE_URL=http://localhost:11434
```

## Examples

```bash
# Start agent mode with a local Ollama model
kdeps serve workflow.yaml

# Use a specific model and system prompt
kdeps serve workflow.yaml --model mistral --system "You are a data analyst."

# Use OpenAI-compatible backend
KDEPS_AGENT_BACKEND=openai KDEPS_AGENT_BASE_URL=https://api.openai.com \
  kdeps serve workflow.yaml --model gpt-4o
```

## Tool dispatch

When the LLM calls a tool, the engine creates a minimal single-resource workflow
targeting that resource and runs it. Tool arguments are injected as query params
and body fields so resources can access them via `get('key')`.

## Differences from workflow mode

| | Workflow mode (`kdeps run`) | Agent mode (`kdeps serve`) |
|---|---|---|
| Execution | DAG, deterministic | LLM loop, tool-driven |
| Entry point | `metadata.targetActionId` | User prompt |
| Resources | Run in declared order | Called as tools on demand |
| Session | Single execution | Interactive REPL |
