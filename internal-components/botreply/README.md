# botreply

> Send a message to a Telegram chat via the Telegram Bot API.

The botreply component calls the Telegram `sendMessage` endpoint with your bot token, target chat ID, and message text. Use it to add Telegram notifications or responses to any workflow.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `token` | string | yes | — | Telegram bot token (from @BotFather) |
| `chatId` | string | yes | — | Telegram chat ID to send the message to |
| `text` | string | yes | — | Message text to send |

## Outputs

Accessed via `output('<your-action-id>')`:

| Field | Type | Description |
|-------|------|-------------|
| `ok` | boolean | `true` when the message was sent successfully |
| `result` | object | Telegram Message object (id, chat, date, text, etc.) |

The output shape is the raw Telegram API response body, parsed as JSON.

## Usage

```yaml
run:
  component:
    name: botreply
    with:
      token: "{{ get('env.TELEGRAM_TOKEN') }}"
      chatId: "{{ get('env.CHAT_ID') }}"
      text: "Deployment complete! Version 1.2.3 is live."
  apiResponse:
    success: true
    response:
      sent: "{{ output('notify').ok }}"
```

Using workflow session values:

```yaml
run:
  component:
    name: botreply
    with:
      token: "{{ get('env.TELEGRAM_TOKEN') }}"
      chatId: "123456789"
      text: "{{ output('prevStep').summary }}"
```

## Requirements

- A valid Telegram bot token. Create a bot via [@BotFather](https://t.me/BotFather).
- The bot must have been started by the target user (or added to the group) before it can send messages.
- The `chatId` can be obtained from the Telegram API `getUpdates` endpoint or a bot like @userinfobot.

## Notes

- Store `token` in an environment variable rather than hardcoding it in the workflow YAML.
- For group chats, prefix the chat ID with `-` (e.g., `-987654321`).
- For channels, use the format `@channelname`.
