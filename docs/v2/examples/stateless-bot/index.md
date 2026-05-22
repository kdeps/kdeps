# Example: Stateless Bot (stdin/stdout)

This workflow runs in workflow mode (`kdeps run`) as a one-shot stdin/stdout bot -- reads a message from stdin (or an env var), calls an LLM, and writes the reply to stdout. Useful for cron jobs, CI pipelines, or custom integrations.

## Files

```
stateless-bot/
├── workflow.yaml
└── resources/
    ├── llm.yaml
    └── reply.yaml
```

## workflow.yaml

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

name: stateless-bot
description: One-shot stdin/stdout bot
version: "1.0.0"
targetActionId: reply
settings:
  agentSettings:
    timezone: Etc/UTC
    installOllama: true
    models:
      - llama3.2:3b

  input:
    sources: [bot]
    bot:
      executionType: stateless
```

## resources/llm.yaml

```yaml

actionId: llm
name: LLM Response
chat:
  messages:
    - role: user
      content: "{{ input('message') }}"
```

## resources/reply.yaml

```yaml

actionId: reply
name: Reply
dependencies:
  - llm

botReply:
  text: "{{ get('llm') }}"
```

## Running

**Via JSON on stdin:**

```bash
echo '{"message":"What is 2+2?","chatId":"42","userId":"u1","platform":"custom"}' \
  | kdeps run workflow.yaml
```

**Via environment variables:**

```bash
export KDEPS_BOT_MESSAGE="What is the capital of France?"
export KDEPS_BOT_PLATFORM="cli"
kdeps run workflow.yaml
```

Output is written to stdout:

```
Paris
```

## See Also

- [Input Sources](../../concepts/input-sources.md) — Full platform config reference
