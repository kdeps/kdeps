# Example: Telegram LLM bot

This workflow is a Telegram bot on a local model. The `telegram_updates` listener starts one run for each message. `llm` writes the answer, and `reply` sends it to the chat with the `telegram_post` connector.

```text
telegram_updates (getUpdates every 2s) -> llm -> reply (telegram_post: sendMessage)
```

```bash
export TELEGRAM_BOT_TOKEN="1234567890:AAH..."   # or: kdeps secret set telegram_bot_token
kdeps run workflow.yaml
```

## Read next

- [Full walkthrough](/reference/examples/telegram-bot/) - every step, file and command
- [Input sources](/workflow/input-sources) - listeners for Telegram, Slack and Discord
- [Connectors](/workflow/resources/connectors) - `telegram_post`, `slack_post`, `discord_post`
