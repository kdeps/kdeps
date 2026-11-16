# Input sources

A workflow gets input from an HTTP server (`apiServer:` under `settings:`), a chat bot, one file, or a terminal prompt. You set the bot, file and interactive sources under `settings:` `input:`.

This sets how a workflow that runs gets its input. Agent mode does not use these sources. The prompt is stdin in the REPL. When the LLM calls a workflow as a tool, that workflow still uses its own input settings.

## Overview

| Source | Use Case |
|--------|----------|
| `api` | Valid source name. Does not open a port. |
| `bot` | Chat bot platforms (Discord, Slack, Telegram) |
| `file` | File content from stdin, env var, or configured path (single-shot) |
| `interactive` | A terminal prompt: each line you type runs the workflow |

An HTTP server starts only when you set `apiServer:` under `settings:`. If you do not set `portNum`, it listens on port 16395. If you do not set `input:`, that server does not start. With no `apiServer`, no `webServer`, no bot source, and no file source, `kdeps run` runs the workflow one time and stops.

## API source

`apiServer:` under `settings:` starts the HTTP server. `sources: [api]` does not.

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

Requests are JSON and routed to resources by `targetActionId:` under the workflow's `metadata:`.

Use `validations.methods` and `validations.routes` in individual resources to scope them to specific routes.

## Bot source

The `bot` source connects to chat platforms. Supported platforms: Discord, Slack, Telegram. WhatsApp has no long-lived connection - Meta calls a webhook - so a WhatsApp bot is an ordinary API workflow: `kdeps new my-bot --template whatsapp-bot`.

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
deploy:
  input:
    sources: [bot]
    bot:
      executionType: polling
      discord: {}   # presence enables the platform; credentials are in ~/.kdeps/config.yaml
      slack: {}
```

Execution types:

- `polling` (default): Long-running persistent connection. Blocks until SIGINT/SIGTERM.
- `stateless`: Reads one message from stdin as JSON, runs the workflow one time, writes the result (the reply) to stdout, then stops.

## File source

The `file` source reads file content from stdin, the `KDEPS_FILE_PATH` environment variable, or a configured path. The workflow runs one time and stops.

```yaml
# workflow.yaml
deploy:
  input:
    sources: [file]
    file:
      path: /data/input.txt
```

Override the path at runtime:

```bash
kdeps run workflow.yaml --file /path/to/document.txt
```

The workflow runs one time and prints its result (the target resource's output) as JSON on stdout, after the progress lines:

```text
✓ Running workflow with file input...
{
  "data": {
    "summary": "..."
  },
  "success": true
}
```

**Size limit.** kdeps reads the full file into memory as `fileContent`. This is
correct for documents.

But a workflow can read a large quantity of data, for
example a large CSV. When a resource copies that content again, memory use can
become much larger than the file size. For example, the resource can put the
content into a `python` component script or an `exec` component argument. Or
a script can make a list of rows in memory.

Above a certain size, the OS can stop the process because it has no more
memory (OOM). This is a `SIGKILL`. It gives no error and no core dump. The
process only stops.

By default, the maximum `--file`/stdin input is 256 MiB.
For a larger file, `kdeps run` stops immediately with a clear error and does
not read the file. To change the limit, set `KDEPS_FILE_INPUT_MAX_BYTES`
(bytes; `0` disables the check):

```bash
KDEPS_FILE_INPUT_MAX_BYTES=1073741824 kdeps run workflow.yaml --file big-export.csv  # 1 GiB
```

If your files are often above this limit, process the file in batches. For
example, use a `python` component or `exec` component step that streams rows.
Do not use one step that keeps the full dataset in memory. The limit is a
safety limit, not a target size.

## Interactive source

`interactive` makes `kdeps run` a prompt. Each line you type runs the target resource with that line as `input('message')` and the prompt's session id as `get('session')`, then the result prints: a chat reply as text, anything else as JSON.

```yaml
# workflow.yaml
deploy:
  input:
    sources: [interactive]   # add api, bot or file to keep those running in the background
repl:
  prompt: "You: "            # text before each line (default "> ")
  sessionId: my-session      # get('session') for every line (default "llm-repl-session")
```

```text
$ kdeps run workflow.yaml
You: What is 12 * 7?
The answer is 84.
You: /quit
```

`/list` shows the resources and components, `/run <actionId> key=value` runs one directly, and `/quit` (or Ctrl+D) exits. `--interactive` opens the same prompt for any workflow. A chat with [`loop:`](/workflow/resources/llm#agent-loop) whose `session` reads `get('session')` keeps one conversation across the lines.

When the target is a `chat` with [`loop:`](/workflow/resources/llm#agent-loop), the prompt is the agent REPL. The REPL has slash commands, the `/model` picker, tool approvals, and saved sessions. Each line goes to the chat as the prompt. The chat's model, `loop.tools`, `loop.maxRounds`, and system `scenario` items set the session. `kdeps --agent-loop` in that folder opens the same REPL.

## Use cases

- Serve a workflow as an HTTP JSON API for a frontend or another service
  (`apiServer:` under `settings:`).
- Run a workflow as a chat bot on Discord, Slack or Telegram (WhatsApp: the `whatsapp-bot` template)
  (`bot`).
- Process one file per invocation from a pipeline, cron job, or CI step
  (`file`).

## See also

- [Workflow configuration](/workflow/configuration) - full `deploy` reference
- [Workflow mode](/workflow/) - how the request-response cycle works
- [Bot configuration](/workflow/configuration#bot-source) - full bot settings
