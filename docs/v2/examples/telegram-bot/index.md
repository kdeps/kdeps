# Example: Telegram LLM bot

This workflow runs with `kdeps run` as a Telegram bot. It polls for messages and replies with a local LLM response (llamafile, with no server install). Two resources: `llm` calls the model, `reply` returns the answer, which kdeps sends back to the chat.

Run it with your bot token:

```bash
export TELEGRAM_BOT_TOKEN="1234567890:AAH..."
kdeps run workflow.yaml
```

## Read next

- [Full walkthrough](/reference/examples/telegram-bot/) - every step, file and command
- [Input sources](../../workflow/input-sources) - Full platform config reference
