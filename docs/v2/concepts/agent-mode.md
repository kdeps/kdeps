# Agent Mode

Agent mode (`kdeps serve`) runs a workflow as a live LLM-driven agent. Every resource defined in the workflow is auto-registered as a callable tool so the LLM can invoke them during a conversation.

See the full reference at [Three Modes - Agent Mode](/modes/agent-mode).

## How it works

1. `kdeps serve workflow.yaml` loads the workflow and builds a tool registry.
2. All workflow resources become tools (tool name = `actionId`).
3. Built-in format tools (JSON, YAML, CSV, XML) are always available.
4. An interactive REPL starts. The LLM receives the user prompt and decides which tools to call.
5. Tool call dispatch runs the target resource with arguments as query params. The result is returned to the LLM.
6. The loop continues until the LLM produces a final answer.

The workflow engine is unchanged. `kdeps run` continues to work exactly as before. Agent mode is additive.

## Command

```bash
kdeps serve workflow.yaml [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--model` | `KDEPS_AGENT_MODEL` or `llama3.2` | LLM model name |
| `--backend` | `KDEPS_AGENT_BACKEND` or `ollama` | LLM backend |
| `--base-url` | `KDEPS_AGENT_BASE_URL` | LLM API base URL |
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

When the LLM calls a tool, kdeps creates a minimal single-resource workflow targeting that resource and runs it. Tool arguments are injected as query params and body fields so resources can access them via `get('key')`.

## Differences from workflow mode

| | Workflow mode (`kdeps run`) | Agent mode (`kdeps serve`) |
|---|---|---|
| Execution | DAG, deterministic | LLM loop, tool-driven |
| Entry point | `metadata.targetActionId` | User prompt |
| Resources | Run in declared order | Called as tools on demand |
| Session | Single execution | Interactive REPL |
