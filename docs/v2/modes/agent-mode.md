# Agent Mode

Agent mode (`kdeps serve`) runs your workflow as a live, interactive LLM agent. Every resource defined in the workflow is automatically registered as a callable tool. The LLM plans and calls them in whatever order is needed to complete the user's request.

Run with:

```bash
kdeps serve workflow.yaml
```

## How it works

1. `kdeps serve workflow.yaml` loads the workflow and builds a tool registry.
2. Each resource's `actionId` becomes a tool name.
3. Built-in format tools (JSON, YAML, CSV, XML) are always available.
4. An interactive REPL starts. You enter a prompt; the LLM decides which tools to call.
5. When the LLM calls a tool, kdeps runs that resource with the tool arguments as query parameters. Results are returned to the LLM.
6. The loop continues until the LLM produces a final answer.

The underlying workflow engine is unchanged. `kdeps run` continues to work exactly as before. Agent mode is additive.

## When to use agent mode

- You want an exploratory, conversational interface over your workflow resources.
- You are prototyping an agent before formalizing the pipeline.
- You want the LLM to dynamically decide which resources to call and in what order.
- You are building a chatbot or assistant that calls your business logic on demand.

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

# Use an OpenAI-compatible backend
KDEPS_AGENT_BACKEND=openai KDEPS_AGENT_BASE_URL=https://api.openai.com \
  kdeps serve workflow.yaml --model gpt-4o
```

## Tool dispatch

When the LLM calls a tool, kdeps creates a minimal single-resource workflow targeting that resource and runs it. Tool arguments become `get('key')` values inside the resource so it can access them normally.

## Differences from workflow mode

| | Workflow mode (`kdeps run`) | Agent mode (`kdeps serve`) |
|---|---|---|
| Execution | DAG, deterministic | LLM loop, tool-driven |
| Entry point | `metadata.targetActionId` | User prompt |
| Resources | Run in declared order | Called as tools on demand |
| Session | Single execution | Interactive REPL |

## See also

- [Workflow Mode](workflow-mode) - Deterministic DAG pipelines
- [MCP Mode](mcp-mode) - Expose resources as MCP tools
- [Tools](../concepts/tools) - Function calling in chat resources
- [CLI Reference](../reference/cli-reference) - `kdeps serve` flags
