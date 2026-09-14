# Input sources

kdeps workflows receive input through three sources: HTTP API, chat bots, and file input. Configure the source in `settings` inside `workflow.yaml`.

*Applies to workflow mode.* This controls how a running workflow is fed. Agent mode does not use these sources; the prompt is stdin in the REPL. When the LLM calls a workflow as a tool, that workflow still uses its own input settings.

## Overview

| Source | Use Case |
|--------|----------|
| `api` | HTTP REST server (default) |
| `bot` | Chat bot platforms (Discord, Slack, Telegram, WhatsApp) |
| `file` | File content from stdin, env var, or configured path (single-shot) |

The default source is `api`. If no input config is specified, the workflow starts an HTTP API server on port 16395.

## API source

The `api` source starts an HTTP REST server. This is the default for all workflows.

```yaml
# workflow.yaml
settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

Requests are JSON and routed to resources based on `metadata.targetActionId` in the workflow.

Use `validations.methods` and `validations.routes` in individual resources to scope them to specific routes.

## Bot source

The `bot` source connects to chat platforms. Supported platforms: Discord, Slack, Telegram, WhatsApp.

Bot credentials belong in `~/.kdeps/config.yaml`, not `workflow.yaml`. The workflow only declares which platforms to enable.

```yaml
# ~/.kdeps/config.yaml
bot_connections:
  discord:
    botToken: "${DISCORD_BOT_TOKEN}"
  slack:
    botToken: "${SLACK_BOT_TOKEN}"
    appToken: "${SLACK_APP_TOKEN}"

# workflow.yaml
settings:
  input:
    sources: [bot]
    bot:
      executionType: polling
      discord: {}   # presence enables the platform; credentials are in ~/.kdeps/config.yaml
      slack: {}
```

Execution types:

- `polling` (default): Long-running persistent connection. Blocks until SIGINT/SIGTERM.
- `stateless`: Reads one message from stdin as JSON, executes the workflow once, writes the reply to stdout, then exits.

## File source

The `file` source reads file content from stdin, the `KDEPS_FILE_PATH` environment variable, or a configured path. The workflow executes once and exits.

```yaml
# workflow.yaml
settings:
  input:
    sources: [file]
    file:
      path: /data/input.txt
```

Override the path at runtime:

```bash
kdeps run workflow.yaml --file /path/to/document.txt
```

**Size limit.** The whole file is read into memory as `fileContent` - fine for
documents, but a workflow that ingests bulk data (a large CSV, say) can spike
memory well past the raw file size once a resource copies that content again
(e.g. interpolating it into a `python:`/`exec:` argument, or a script that
builds an in-memory row list). Past a certain size that risks an OOM kill from
the OS - which, being a `SIGKILL`, leaves no error and no core dump, just a
vanished process. `--file`/stdin input is capped at **256 MiB** by default;
`kdeps run` fails fast with a clear error instead of reading a file over that.
Raise or lower it with `KDEPS_FILE_INPUT_MAX_BYTES` (bytes; `0` disables the
check):

```bash
KDEPS_FILE_INPUT_MAX_BYTES=1073741824 kdeps run workflow.yaml --file big-export.csv  # 1 GiB
```

If you hit this limit often, prefer having the workflow process the file in
batches (e.g. a `python:`/`exec:` step that streams rows) over one step that
holds the whole dataset in memory - the limit is a safety net, not a size to
build up to.

## Interactive REPL

Start an interactive LLM REPL alongside the normal workflow execution with the `--interactive` flag. This is independent of the configured input source:

```bash
kdeps run workflow.yaml --interactive
```

This opens a terminal REPL where you can send prompts and invoke tools interactively.

## Use cases

- Serve a workflow as an HTTP JSON API for a frontend or another service
  (`api`).
- Run a workflow as a chat bot on Discord, Slack, Telegram, or WhatsApp
  (`bot`).
- Process one file per invocation from a pipeline, cron job, or CI step
  (`file`).

## See also

- [Workflow configuration](/workflow/configuration) - full `settings` reference
- [Workflow mode](/workflow/) - how the request-response cycle works
- [Bot configuration](/workflow/configuration#bot-source) - full bot settings
