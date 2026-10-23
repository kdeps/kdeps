# Telegram Bot Example

A Telegram bot that receives user messages and replies using a local LLM (`llama3.2:1b` llamafile, no server install needed).

## Features

- Polls Telegram for new messages every second
- Sends each message to a local LLM with a custom persona ("Kodi")
- Replies to the user automatically on Telegram

## Prerequisites

- A Telegram bot token — create one with [@BotFather](https://t.me/botfather) (`/newbot`)
- Docker or a local kdeps runtime

## Setup

### 1. Get a Bot Token

Talk to [@BotFather](https://t.me/botfather) on Telegram:

```
/newbot
→ Name: My Kodi Bot
→ Username: my_kodi_bot
→ Token: 1234567890:AAH...
```

### 2. Run the Bot

```bash
export TELEGRAM_BOT_TOKEN="1234567890:AAH..."

# From the examples/telegram-bot directory
kdeps run workflow.yaml

# Or from the project root
kdeps run examples/telegram-bot/workflow.yaml
```

You should see:

```
Bot input sources active:
  • Telegram (polling)

Starting bot runners... (press Ctrl+C to stop)
```

### 3. Chat with the Bot

Open Telegram, find your bot by its username, and send it a message. It will reply using the LLM.

## Structure

```
telegram-bot/
├── workflow.yaml              # Bot source config, Telegram credentials; targetActionId: reply
└── resources/
    ├── llm.yaml               # LLM chat resource - receives input('message'), applies persona
    └── reply.yaml             # apiResponse - its response is sent back to the chat
```

kdeps sends the workflow's result (the `reply` resource's `apiResponse` response) to the chat the message came from.

## Key Expressions

| Expression | Description |
|------------|-------------|
| `input('message')` | The text the Telegram user sent |
| `input('chatId')` | Telegram chat ID (for targeted replies) |
| `input('userId')` | Telegram user ID |
| `input('platform')` | Always `"telegram"` for this bot |
| `get('llm').message.content` | The LLM-generated reply text |

## Customization

### Use a Different Model

```yaml
# workflow.yaml
agentSettings:
  models:
    - llama3.1:8b     # Smarter, slower

# resources/llm.yaml
chat:
  model: llama3.1:8b
```

### Change the Persona

Edit the `scenario` block in `resources/llm.yaml`:

```yaml
scenario:
  - role: assistant
    prompt: |
      You are Aria, a customer support agent for Acme Corp.
      Help users with order status, returns, and product questions.
```

### Add Discord Support

Add a `discord` block to `workflow.yaml`:

```yaml
# workflow.yaml
bot:
  executionType: polling
  discord: {}    # presence enables the platform
  telegram: {}
```

Credentials live in `~/.kdeps/config.yaml`, not in the workflow:

```yaml
bot_connections:
  discord:
    botToken: "${DISCORD_BOT_TOKEN}"
  telegram:
    botToken: "${TELEGRAM_BOT_TOKEN}"
```

## See Also

- [Stateless Bot Example](../stateless-bot/) — One-shot stdin/stdout execution
- [Input Sources Documentation](../../docs/v2/concepts/input-sources.md)
- [Bot Tutorial](../../docs/v2/concepts/input-sources.md)
- [Voice Assistant Example](../voice-assistant/) — Audio/microphone input with TTS output
