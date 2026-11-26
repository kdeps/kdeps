# Input sources

A workflow gets input from an HTTP server (`apiServer:` under `deploy:`), a chat bot, one file, a terminal prompt, or listeners. A listener starts one run for each event: a new line in a file or a named pipe, an item that a polled tool returns, or a webhook POST.

```yaml
# workflow.yaml
deploy:
  input:
    sources: [bot]     # bot, file or interactive; apiServer: starts the HTTP server
    listen:            # one run per event, the event is the input: input('line')
      - file: { path: inbox.txt }
```

## Read next

- [Input sources reference](/reference/workflow/input-sources) - every key, option and detail
- [Workflow configuration](/workflow/configuration) - full `deploy` reference
- [Workflow](/workflow/) - the request and the response
