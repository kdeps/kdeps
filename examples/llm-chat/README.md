# llm-chat

Interactive LLM chat: `kdeps run` opens a prompt, and each line you type runs the workflow.

## Interactive prompt

```yaml
# workflow.yaml
deploy:
  input:
    sources: [interactive]   # each typed line runs targetActionId
repl:
  prompt: "You: "            # text before each line (default "> ")
  sessionId: "my-session"    # session id for every line (default "llm-repl-session")
```

```bash
kdeps run examples/llm-chat/workflow.yaml
```

```text
You: hello
Hi! How can I help you today?
You: what is 2+2?
2 + 2 = 4.
You: /quit
Goodbye!
```

`/quit`, `/exit` or Ctrl+D stop the prompt.

## Prompt and API together

`sources: [api, interactive]` with an `apiServer` keeps the HTTP API running while you type.

```bash
curl -s -X POST http://localhost:16400/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'
```

## Reading the line

The typed text is `input('message')`, and the session id is `get('session')`:

```yaml
chat:
  model:
    name: llama3.2:1b
  context:
    role: assistant
    prompt: "{{ input('message') }}"
```
