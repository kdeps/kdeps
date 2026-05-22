# Input Sources

kdeps workflows receive input through three sources: HTTP API, chat bots, and file input. Configure the source in `settings` inside `workflow.yaml`.

## Overview

| Source | Use Case |
|--------|----------|
| `api` | HTTP REST server (default) |
| `bot` | Chat bot platforms (Discord, Slack, Telegram, WhatsApp) |
| `file` | File content from stdin, env var, or configured path (single-shot) |

The default source is `api`. If no input config is specified, the workflow starts an HTTP API server on port 16395.

## API Source

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

## Bot Source

The `bot` source connects to chat platforms. Supported platforms: Discord, Slack, Telegram, WhatsApp.

```yaml
# workflow.yaml
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

## Interactive REPL

Start an interactive LLM REPL alongside the normal workflow execution with the `--interactive` flag. This is independent of the configured input source:

```bash
kdeps run workflow.yaml --interactive
```

This opens a terminal REPL where you can send prompts and invoke tools interactively.

## See Also

- [Workflow Configuration](../configuration/workflow) - Full `settings` reference
- [Workflow Mode](../modes/workflow-mode) - How the request-response cycle works
- [Bot Configuration](../configuration/workflow#bot-source) - Full bot settings
