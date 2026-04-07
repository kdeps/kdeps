# llm-chat

Interactive LLM chat example demonstrating the `sources: [llm]` input source.

## Overview

The `llm` input source supports two execution types:

| `executionType` | Behaviour |
|---|---|
| `stdin` (default) | Interactive REPL loop — read message from stdin, run workflow, print LLM reply |
| `apiServer` | Start the HTTP API server so REST clients can drive the chat |

## Stdin REPL Mode

```yaml
input:
  sources: [llm]
  llm:
    executionType: stdin
    prompt: "You: "          # text printed before each read (default: "> ")
    sessionId: "my-session"  # session ID shared across all turns (default: "llm-repl-session")
```

Run:

```bash
kdeps run examples/llm-chat/workflow.yaml
```

Output:

```
  ✓ Starting LLM interactive REPL (type /quit or /exit to stop, Ctrl+D for EOF)

You: hello
Hi! How can I help you today?
You: what is 2+2?
2 + 2 = 4.
You: /quit
Goodbye!
```

Commands:
- `/quit` or `/exit` — exit the REPL cleanly
- Ctrl+D — send EOF to stop

## API Server Mode

```yaml
input:
  sources: [llm]
  llm:
    executionType: apiServer

# Required for apiServer mode:
apiServerMode: false  # ignored — apiServer mode is set by executionType
portNum: 16400
```

When `executionType: apiServer`, kdeps starts the HTTP API server just like
`apiServerMode: true`. Configure `apiServer.routes` and `portNum` as usual.

Run:

```bash
kdeps run examples/llm-chat/workflow.yaml
# Then POST to the configured route:
curl -s -X POST http://localhost:16400/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'
```

## Resource: input("message")

In either mode the user's text is available via `input("message")`:

```yaml
run:
  chat:
    model: llama3.2:1b
    role: assistant
    prompt: "{{ input('message') }}"
```

## Session Context

In stdin mode a single `sessionId` is shared across all turns so the LLM
retains multi-turn context. Configure it in `input.llm.sessionId`. In
apiServer mode session cookies are set by the HTTP layer as usual.

## Configuration Reference

```yaml
input:
  sources: [llm]
  llm:
    executionType: stdin     # "stdin" (default) or "apiServer"
    prompt: "> "             # REPL prompt text (stdin mode only)
    sessionId: "my-session"  # fixed session ID (stdin mode only)
```
