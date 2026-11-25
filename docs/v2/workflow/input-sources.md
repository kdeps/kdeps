# Input sources

A workflow gets input from an HTTP server (`apiServer:` under `deploy:`), a chat bot, one file, or a terminal prompt. You set the bot, file and interactive sources under `input:` in `deploy:`.

```yaml
# workflow.yaml
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

## Read next

- [Input sources reference](/reference/workflow/input-sources) - overview, API source, bot source, file source, interactive source, use cases
- [Workflow configuration](/workflow/configuration) - full `deploy` reference
- [Workflow](/workflow/) - the request and the response
- [Bot configuration](/workflow/configuration#bot-source) - full bot settings
