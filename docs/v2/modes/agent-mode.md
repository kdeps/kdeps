# Agent Mode

Agent mode (`kdeps serve`) turns your workflow into an interactive LLM loop. Every resource is auto-registered as a callable tool. The LLM decides which tools to call and in what order based on the user's prompt.

Run with:

```bash
kdeps serve workflow.yaml
```

## How it works

```
you type a prompt
        |
        v
+-------------------------+
|  LLM receives prompt    |  <- model set by --model flag or KDEPS_AGENT_MODEL
|  + tool registry        |  <- one tool per actionId in workflow.yaml
+-------------------------+
        |  LLM decides to call tool "fetch"
        v
+-------------------------+
|  kdeps runs resource    |  <- tool args become get('key') inside the resource
|  actionId: fetch        |
+-------------------------+
        |  result returned to LLM
        v
+-------------------------+
|  LLM continues loop     |  <- may call more tools or produce final answer
+-------------------------+
        |
        v
   final answer printed
```

The same workflow.yaml runs in both modes. `kdeps run` is unchanged -- agent mode is additive.

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

When the LLM calls a tool, kdeps creates a minimal single-resource workflow targeting that resource and runs it. Tool arguments become `get('key')` values inside the resource -- the same way request parameters work in workflow mode. The resource does not need any special wiring to work as a tool.

## Differences from workflow mode

| | Workflow mode (`kdeps run`) | Agent mode (`kdeps serve`) |
|---|---|---|
| Execution | DAG, deterministic | LLM loop, tool-driven |
| Entry point | `metadata.targetActionId` | User prompt |
| Resources | Run in declared order | Called as tools on demand |
| Session | Single execution | Interactive REPL |

## See also

- [Workflow Mode](workflow-mode) - Deterministic DAG pipelines
- [Tools](../concepts/tools) - Function calling in chat resources
- [CLI Reference](../reference/cli-reference) - `kdeps serve` flags
