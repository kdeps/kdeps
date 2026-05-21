# Example: Telegram LLM Bot

A minimal ready-to-run kdeps workflow that connects a Telegram bot to a local Ollama LLM.

## Files

```
telegram-bot/
├── workflow.yaml
└── resources/
    ├── llm.yaml
    └── reply.yaml
```

## workflow.yaml

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

name: telegram-llm-bot
description: Telegram bot that answers messages with an LLM
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
      executionType: polling
      telegram:
        botToken: "{{ env('TELEGRAM_BOT_TOKEN') }}"
        pollIntervalSeconds: 1
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

```bash
export TELEGRAM_BOT_TOKEN="1234567890:AAH..."
kdeps run workflow.yaml
```

## See Also

- [Input Sources](../../concepts/input-sources.md) — Full platform config reference
