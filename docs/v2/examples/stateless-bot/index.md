# Example: Stateless Bot (stdin/stdout)

A one-shot kdeps workflow that reads a bot message from stdin (or environment variables), runs an LLM, and writes the reply to stdout. Useful for cron jobs, CI pipelines, or custom integrations.

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

metadata:
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
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: llm
  name: LLM Response

run:
  chat:
    backend: ollama
    model: llama3.2:3b
    messages:
      - role: user
        content: "{{ input('message') }}"
```

## resources/reply.yaml

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: reply
  name: Reply

dependencies:
  - llm

run:
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

- [Bot Tutorial](../../tutorials/bot.md) — Step-by-step walkthrough
- [Input Sources](../../concepts/input-sources.md) — Full platform config reference
