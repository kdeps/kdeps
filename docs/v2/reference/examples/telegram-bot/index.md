# Example: Telegram LLM bot: full text

This page has every step and detail. The [overview](/examples/telegram-bot/) is the short version.

The `telegram_updates` listener polls the Bot API and starts one workflow run for each message. `llm` answers with a local model (llamafile, no server install). `reply` sends the answer to the same chat with the `telegram_post` connector.

## Files

```text
telegram-bot/
├── workflow.yaml
└── resources/
    ├── llm.yaml
    └── reply.yaml
```

## workflow.yaml

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: telegram-bot
  description: Telegram bot that answers messages with a local LLM
  version: "1.0.0"
  targetActionId: reply

deploy:
  runtime:
    timezone: Etc/UTC

  input:
    listen:
      # Polls getUpdates every 2s. Token: TELEGRAM_BOT_TOKEN or the vault secret telegram_bot_token.
      - listener: telegram_updates
```

## resources/llm.yaml

<div v-pre>

```yaml
actionId: llm
name: LLM Response

chat:
  model:
    name: llama3.2:1b
  context:
    prompt: "{{ input('message').text }}"   # the event is one Telegram update
```

</div>

## resources/reply.yaml

<div v-pre>

```yaml
actionId: reply
name: Reply
flow:
  requires:
    - llm

# Send the answer to the chat the message came from (Bot API sendMessage).
component:
  name: telegram_post
  with:
    path: sendMessage
    body:
      chat_id: "{{ input('message').chat.id }}"
      text: "{{ get('llm').message.content }}"
```

</div>

## Running

```bash
export TELEGRAM_BOT_TOKEN="1234567890:AAH..."   # or: kdeps secret set telegram_bot_token
kdeps run workflow.yaml
# [listen telegram_updates] {"ok":true,"result":{...}}
```

The listener keeps its place in `~/.kdeps/projects/<folder>-<hash>/listen/telegram_updates.cursor`, so a restart does not answer old messages again.

## See also

- [Input sources](/workflow/input-sources) - listeners for Telegram, Slack and Discord
- [Connectors](/workflow/resources/connectors) - `telegram_post`, `slack_post`, `discord_post`
