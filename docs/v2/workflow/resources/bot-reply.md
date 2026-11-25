# Bot replies

kdeps sends a bot workflow's result back to the chat that sent the message: Discord, Slack, Telegram, or stdout. There is no separate reply resource.

```text
message -> workflow (llm -> reply) -> result text -> same chat
```

```yaml
# workflow.yaml (abridged)
metadata:
  targetActionId: reply        # its result is the reply
deploy:
  input:
    sources: [bot]
    bot:
      executionType: polling
      telegram: {}   # presence enables the platform; credentials are in ~/.kdeps/config.yaml bot_connections
```

## Read next

- [Bot replies reference](/reference/resources/bot-reply) - every key, option and detail
- [Input sources](/workflow/input-sources) - configure `bot:` under `deploy:` `input:`
- [API response](/workflow/resources/api-response) - the `apiResponse:` block
