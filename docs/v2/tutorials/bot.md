# Tutorial: Build a Telegram Bot with LLM Replies

This tutorial walks through creating a Telegram bot that replies to every message using an LLM running locally via Ollama. The same pattern applies to Discord, Slack, and WhatsApp with minor config changes.

## Prerequisites

- kdeps CLI installed (`kdeps version`)
- Docker (for running the agent container)
- A Telegram bot token — create one with [@BotFather](https://t.me/botfather) (`/newbot`)
- Ollama installed locally, or `installOllama: true` in `agentSettings`

---

## Step 1 — Create the Workflow File

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
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

Key points:
- `sources: [bot]` enables the bot input subsystem
- `executionType: polling` keeps the process running and polls Telegram for new messages
- `botToken` uses the `env()` expression so the token is never hard-coded
- `targetActionId: reply` — the workflow ends by executing the `reply` resource

---

## Step 2 — Create the LLM Resource

```yaml
# resources/llm.yaml
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

`input('message')` retrieves the text the Telegram user sent.

---

## Step 3 — Create the Reply Resource

The `botReply` resource sends the text back to the originating platform. In polling mode it calls the platform's API; in stateless mode it writes to stdout. The dispatcher loop continues after this resource returns — no explicit restart is needed.

```yaml
# resources/reply.yaml
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

---

## Step 4 — Run the Bot

Export your bot token, then start the workflow:

```bash
export TELEGRAM_BOT_TOKEN="1234567890:AAH..."

kdeps run workflow.yaml
```

You should see:

```
Bot input sources active:
  • Telegram (polling)

Starting bot runners... (press Ctrl+C to stop)
```

Send a message to your bot in Telegram — it replies with the LLM's answer.

---

## Step 5 — Add a System Prompt (Optional)

Give your bot a persona by adding a `scenario` block to the LLM resource:

```yaml
run:
  chat:
    backend: ollama
    model: llama3.2:3b
    scenario:
      - role: assistant
        prompt: |
          You are Kodi, a helpful AI assistant.
          Keep your answers short and friendly.
          Always respond in the same language as the user.
    messages:
      - role: user
        content: "{{ input('message') }}"
```

---

## Stateless Mode

Stateless mode runs the workflow exactly once from a shell command — no long-running process needed. Useful for cron jobs, CI pipelines, or custom integrations.

Change `executionType` to `stateless`:

```yaml
settings:
  input:
    sources: [bot]
    bot:
      executionType: stateless
```

Then pipe a JSON message to `kdeps run`:

```bash
echo '{"message":"What is 2+2?","chatId":"42","userId":"u1","platform":"custom"}' \
  | kdeps run workflow.yaml
```

Output (stdout):

```
4
```

Or use environment variables instead of JSON:

```bash
export KDEPS_BOT_MESSAGE="What is the capital of France?"
export KDEPS_BOT_PLATFORM="cli"
kdeps run workflow.yaml
```

---

## Adding More Platforms

Extend the `bot` block to run on Discord and Telegram simultaneously:

```yaml
settings:
  input:
    sources: [bot]
    bot:
      executionType: polling
      discord:
        botToken: "{{ env('DISCORD_BOT_TOKEN') }}"
      telegram:
        botToken: "{{ env('TELEGRAM_BOT_TOKEN') }}"
```

The same workflow resources receive messages from both platforms. Use `input('platform')` to branch if needed:

```yaml
# resources/reply.yaml
run:
  botReply:
    text: |
      {{ if eq (input('platform')) "discord" }}
      **{{ get('llm') }}**
      {{ else }}
      {{ get('llm') }}
      {{ end }}
```

---

## Full Directory Structure

```
telegram-llm-bot/
├── workflow.yaml
└── resources/
    ├── llm.yaml
    └── reply.yaml
```

---

## See Also

- [Input Sources](../concepts/input-sources.md) — All bot platform configs and field reference
- [Telegram Bot Example](../../../examples/telegram-bot/) — Ready-to-run example
- [Stateless Bot Example](../../../examples/stateless-bot/) — One-shot stdin/stdout example
- [LLM Resource](../resources/llm) — Chat, scenario, backend options
