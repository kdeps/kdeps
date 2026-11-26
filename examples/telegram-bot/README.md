# Telegram Bot Example

A Telegram bot that answers messages with a local LLM (`llama3.2:1b`).

```text
telegram_updates listener -> llm -> reply (telegram_post: sendMessage)
```

- The `telegram_updates` listener polls `getUpdates` every 2 seconds. Each message starts one workflow run.
- `llm` answers the text with the "Kodi" persona.
- `reply` sends the answer to the same chat with the `telegram_post` component.

## Run

1. Create a bot with [@BotFather](https://t.me/botfather) (`/newbot`) and copy the token.
2. Give kdeps the token:
   ```bash
   export TELEGRAM_BOT_TOKEN="1234567890:AAH..."
   # or keep it in the vault: kdeps secret set telegram_bot_token
   ```
3. Run the workflow:
   ```bash
   kdeps run examples/telegram-bot/workflow.yaml
   ```
4. Send your bot a message on Telegram.

Each run prints one line: `[listen telegram_updates] {...}`.

## Files

```text
telegram-bot/
├── workflow.yaml        # input.listen: telegram_updates; targetActionId: reply
└── resources/
    ├── llm.yaml         # prompt: input('message').text
    └── reply.yaml       # telegram_post sendMessage to input('message').chat.id
```

## Event fields

| Expression | Value |
|---|---|
| `input('message').text` | The text the user sent |
| `input('message').chat.id` | The chat to answer |
| `input('message').from.id` | The sender's user ID |
| `input('update_id')` | The update number |

## Slack or Discord

Use the `slack_events` listener with `slack_post`, or `discord_gateway` with `discord_post`. See [Input sources](../../docs/v2/workflow/input-sources.md) and [Connectors](../../docs/v2/workflow/resources/connectors.md).
