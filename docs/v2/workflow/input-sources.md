# Input sources

A workflow gets input from an HTTP server (`apiServer:` under `deploy:`), one file, a terminal prompt, or listeners. A chat bot is a listener, and its reply is a connector call. A listener starts one run for each event: a new line in a file or a named pipe, an item that a polled tool returns, a webhook POST, or a websocket message.

```yaml
# workflow.yaml
deploy:
  input:
    listen:            # one run per event; the event is the input
      - file: { path: inbox.txt }      # input('line')
      - listener: telegram_updates     # ready-made: also slack_events, discord_gateway
```

`sources: [file]` reads one file and stops. `sources: [interactive]` runs the workflow for each line you type.

## Read next

- [Input sources reference](/reference/workflow/input-sources) - every key, option and detail
- [Workflow configuration](/workflow/configuration) - full `deploy` reference
- [Workflow](/workflow/) - the request and the response
