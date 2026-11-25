# Input sources

A workflow gets input from an HTTP server (`apiServer:` under `deploy:`), a chat bot, one file, or a terminal prompt. You set the bot, file and interactive sources under `input:` in `deploy:`.

```yaml
# workflow.yaml
deploy:
  input:
    sources: [bot]     # bot, file or interactive; apiServer: starts the HTTP server
```

## Read next

- [Input sources reference](/reference/workflow/input-sources) - every key, option and detail
- [Workflow configuration](/workflow/configuration) - full `deploy` reference
- [Workflow](/workflow/) - the request and the response
