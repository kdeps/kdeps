# Input Sources

KDeps workflows receive input through three sources: HTTP API, chat bots, and file input. Sources are configured in the `settings.input` block of your `workflow.yaml`.

## Overview

| Source | Use Case |
|--------|----------|
| `api` | HTTP API requests (default, REST/JSON) |
| `bot` | Chat bot platforms (Discord, Slack, Telegram, WhatsApp) |
| `file` | File content from stdin, env var, or configured path (single-shot) |

The default source is `api` — if no input config is specified, the workflow starts an HTTP API server on port 16395.

## API Source

The `api` source starts an HTTP API server. This is the default for all workflows.

```yaml
settings:
  input:
    sources: [api]
  apiServerMode: true
  apiServer:
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

Requests are JSON and routed to resources based on the `targetActionId` in workflow metadata.

## Bot Source

The `bot` source connects to chat platforms. Supported platforms: Discord, Slack, Telegram, WhatsApp.

```yaml
settings:
  input:
    sources: [bot]
    bot:
      executionType: polling
      discord:
        botToken: "${DISCORD_BOT_TOKEN}"
      slack:
        botToken: "${SLACK_BOT_TOKEN}"
        appToken: "${SLACK_APP_TOKEN}"
```

Execution types:
- `polling` (default): Long-running persistent connection. Blocks until SIGINT/SIGTERM.
- `stateless`: Reads one message from stdin as JSON, executes the workflow once, writes the reply to stdout, then exits.

## File Source

The `file` source reads file content from stdin, the `KDEPS_FILE_PATH` environment variable, or a configured path. The workflow executes once and exits.

```yaml
settings:
  input:
    sources: [file]
    file:
      path: /data/input.txt
```

Use `--file` on the CLI to override the path:
```bash
kdeps run workflow.yaml --file /path/to/document.txt
```

## LLM REPL

The interactive LLM REPL is started with the `--interactive` flag, regardless of configured input source:

```bash
kdeps run workflow.yaml --interactive
```

This opens a terminal REPL where you can chat with the LLM and invoke tools, components, and agents interactively.
