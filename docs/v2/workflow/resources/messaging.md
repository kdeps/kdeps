# Messaging resources

A workflow sends a message with a connector component. A listener receives the message; the connector sends the answer.

| Component | Channel | Reference |
| :--- | :--- | :--- |
| `slack_post` | Slack Web API (`chat.postMessage`) | [Connectors](/workflow/resources/connectors) |
| `telegram_post` | Telegram Bot API (`sendMessage`) | [Connectors](/workflow/resources/connectors) |
| `discord_post` | Discord REST API (channel messages) | [Connectors](/workflow/resources/connectors) |

## See also

- [Telegram bot](/examples/telegram-bot/) - a listener in, `telegram_post` out
- [Input sources](/workflow/input-sources) - `listen:` with `telegram_updates`, `slack_events`, `discord_gateway`
- [Secrets](/workflow/secrets) - keep the bot tokens in the vault
- [Resources overview](/workflow/resources) - all resource types
