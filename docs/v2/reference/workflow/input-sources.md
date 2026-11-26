# Input sources reference

This page lists every field and option. The [overview](/workflow/input-sources) shows what it does and a first example.


A workflow gets input from an HTTP server (`apiServer:` under `deploy:`), one file, a terminal prompt, or listeners. You set the file and interactive sources and `listen:` under `input:` in `deploy:`. A chat bot is a listener in and a connector out: see [Ready-made listeners](#ready-made-listeners).

This sets how a run gets its input. `interactive` is one of the three sources.

## Overview

| Source | Use Case |
|--------|----------|
| `api` | Valid source name. Does not open a port. |
| `file` | File content from stdin, env var, or configured path (single-shot) |
| `interactive` | A terminal prompt: each line you type runs the workflow |
| `listen:` | Listeners: one run for each new line in a file or a named pipe, each item a polled tool returns, each webhook POST, or each websocket message a rule emits |

An HTTP server starts only when you set `apiServer:` under `settings:`. If you do not set `portNum`, it listens on port 16395. If you do not set `input:`, that server does not start. With no `apiServer`, no `webServer`, no file source and no `listen:`, `kdeps run` runs the workflow one time and stops.

## API source

`apiServer:` under `deploy:` starts the HTTP server. `sources: [api]` does not.

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
    sources: [interactive]   # add api or file to keep those running in the background
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

`/list` shows the resources and components, `/run <actionId> key=value` runs one directly, and `/quit` (or Ctrl+D) exits. `--interactive` opens the same prompt for any workflow. A chat with [`loop:`](/reference/resources/llm#agent-loop) whose `session` reads `get('session')` keeps one conversation across the lines.

When the target is a `chat` with `loop:`, each line goes to that chat as the prompt, and the chat runs as the [agent loop](/workflow/agent-loop/).

## Listeners

`listen:` starts one workflow run for each event. The event is the run's input. `kdeps run` keeps running until you stop it. With `apiServer:` or `webServer:`, the listeners run next to the server.

```d2
direction: right
file: "file: new line" {shape: oval}
fifo: "fifo: line in a named pipe" {shape: oval}
poll: "poll: tool reply item" {shape: oval}
hook: "webhook: signed POST" {shape: oval}
ws: "websocket: message a rule emits" {shape: oval}
q: "queue per key"
run: "one workflow run\ninput('line'), input('id')"
file -> q
fifo -> q
poll -> q
hook -> q
ws -> q
q -> run
```

<div v-pre>

```yaml
# workflow.yaml
deploy:
  input:
    listen:
      - id: inbox                      # names the listener in logs and its saved position
        file:
          path: /data/inbox.ndjson     # followed like tail -f; each new line is one run
          framing: json                # line (default): input('line'); json: the object's fields
      - id: pipe
        fifo:
          path: /run/kdeps/in          # named pipe, created when missing (Linux and macOS)
      - id: notifications
        key: "{{.event.repository.full_name}}"   # same key: in order; other keys: in parallel
        poll:
          tool: github_get             # any kdeps API tool
          with:
            path: "notifications?since={{.cursor}}"
          every: 60s                   # default 30s, minimum 1s
          items: ""                    # dotted path to the list in the reply; empty: the reply
          cursor: "{{.item.updated_at}}"   # saved after each item, read back as .cursor
          start: "2026-01-01T00:00:00Z"    # .cursor before anything is saved
      - id: github
        webhook:
          addr: ":8090"                # default :16396; webhooks on one addr share a server
          path: /hooks/github
          wait: false                  # true: run first, answer with the result (200)
          verify:                      # a POST without a matching HMAC gets 401
            secret: [GITHUB_WEBHOOK_SECRET, "secret:github_webhook_secret"]
            message: "{{.body}}"       # default
            signature: '{{trimPrefix (index .headers "X-Hub-Signature-256") "sha256="}}'
```

</div>

| Key | Meaning |
|---|---|
| `id` | Listener name. Default `listen-<index>`. Letters, digits, `_`, `.` and `-`. |
| `key` | A template over `.event`. Events with the same key run in order; different keys run in parallel. Empty: every event of the listener runs in order. |
| `file.path`, `fifo.path` | The file to follow, or the named pipe to read. |
| `framing` | `line` (default): the event is `{"line": "..."}`. `json`: each line is a JSON object, and its fields are the event. |
| `poll.tool` | A kdeps API tool, such as a [connector](/reference/resources/connectors). Its reply must be JSON. |
| `poll.with` | The tool arguments. Each string is a template over `.cursor`. |
| `poll.items` | A dotted path to the event list in the reply (`result`, `data.items`). A single object is one event. |
| `poll.cursor`, `poll.start` | The next cursor, a template over `.item` and `.cursor`, and the first cursor. |
| `webhook.path`, `webhook.addr` | The URL path and the listen address. A JSON body is the event; a form body is its fields; another body is `{"body": "..."}`. |
| `webhook.wait` | `false` (default): answer `202` at once and run after. `true`: run, then answer with the result, or `500` and the error. |
| `webhook.handshake` | A template over `.event`, `.body`, `.headers`, `.query` and `.method`. When it renders text, kdeps answers with that text and starts no run. |
| `webhook.verify` | An HMAC check: `secret` (env names and `secret:NAME` entries), `algorithm` (`sha256` or `sha1`), `encoding` (`hex` or `base64`), `message` and `signature` templates. |

### Ready-made listeners

`listener:` names a listener asset: one transport block in YAML, installed on first use. Reply with the service's connector in a step.

<div v-pre>

```yaml
# workflow.yaml
deploy:
  input:
    listen:
      - listener: telegram_updates     # or slack_events, discord_gateway
        key: "{{.event.message.chat.id}}"   # one chat in order, other chats in parallel

# resources/reply.yaml
actionId: reply
component:
  name: telegram_post
  with:
    path: sendMessage
    body:
      chat_id: "{{ input('message').chat.id }}"
      text: "{{ output('answer') }}"
```

</div>

| Listener | Transport | Token | The event | Reply with |
|---|---|---|---|---|
| `telegram_updates` | poll `getUpdates` every 2s | `TELEGRAM_BOT_TOKEN`, `TELEGRAM_TOKEN`, vault `telegram_bot_token` | the update: `input('message').text`, `input('message').chat.id` | `telegram_post` (`sendMessage`) |
| `slack_events` | Socket Mode websocket, no public URL | `SLACK_APP_TOKEN` (`xapp-`), vault `slack_app_token` | the message event: `input('text')`, `input('channel')`, `input('user')` | `slack_post` (`chat.postMessage`) |
| `discord_gateway` | Gateway websocket, heartbeat and identify | `DISCORD_BOT_TOKEN`, vault `discord_bot_token` | the message: `input('content')`, `input('channel_id')`, `input('author').username` | `discord_post` (`channels/ID/messages`) |

Messages from bots start no run, so a reply does not start a loop. `slack_events` acknowledges every envelope. `discord_gateway` needs the Message Content intent turned on for the bot. Each asset is a plain listener block: copy it into `listen:` to change it.

### Websocket

A `websocket` listener keeps one connection open and connects again after a failure (wait 1s, doubling up to 1m). `on:` rules describe the service's protocol in YAML: kdeps checks every rule for each message, in order. A rule matches when `when` renders text other than `false`.

<div v-pre>

```yaml
# a gateway that says hello, wants an identify, and a heartbeat with the last sequence
- id: gateway
  websocket:
    env:
      TOKEN: [BOT_TOKEN, "secret:bot_token"]     # .env.TOKEN; the listener does not start without it
    url: "wss://gateway.example.com/?v=10"
    headers: { User-Agent: kdeps }
    heartbeat:
      send: '{"op":1,"d":{{or .state.seq "null"}}}'   # starts when a rule sets every:
    on:
      - when: '{{if eq (int .msg.op) 10}}y{{end}}'     # hello
        every: '{{.msg.d.heartbeat_interval}}ms'
        send: '{"op":2,"d":{"token":"{{.env.TOKEN}}"}}'
      - when: '{{.msg.s}}'
        set: { seq: '{{.msg.s}}' }                     # kept in .state across messages and reconnects
      - when: '{{if eq .msg.t "MESSAGE_CREATE"}}y{{end}}'
        emit: '{{json .msg.d}}'                        # one run; JSON gives the event fields
      - when: '{{if eq (int .msg.op) 7}}y{{end}}'
        reconnect: true

# a service that hands out the URL and wants each envelope acknowledged
- id: socket
  websocket:
    open: { tool: socket_open }                        # any kdeps API tool; its JSON reply is .open
    url: "{{.open.url}}"
    on:
      - when: '{{.msg.envelope_id}}'
        send: '{"envelope_id":"{{.msg.envelope_id}}"}'
      - when: '{{if eq .msg.type "events_api"}}y{{end}}'
        emit: '{{json .msg.payload.event}}'
```

</div>

| Key | Meaning |
|---|---|
| `env` | Names for `.env`: each lists environment variables and `secret:NAME` vault entries. |
| `open` | A kdeps API tool called before each connect; its JSON reply is `.open`. |
| `url`, `headers` | The `ws://` or `wss://` address and the connect headers. |
| `send` | Messages sent after each connect. |
| `heartbeat` | `send` on a timer; `every` sets the first interval, and a rule's `every` sets or changes it. |
| `on[].when` | The rule matches when this renders text other than `false`. Templates see `.msg` (JSON), `.raw` (text), `.state`, `.env` and `.open`. |
| `on[].set`, `every`, `send`, `emit`, `reconnect` | Run in this order: store values in `.state`, set the heartbeat, send a message, start one run, connect again. |

Listener templates use the same helpers as tool `http:` blocks (`json`, `get`, `add`, `int`, `urlquery`, `kv`, ...), not the `input()` expressions of a resource.

A handshake answers a service's URL check. Only a POST can start a run, and `verify` applies to each POST:

<div v-pre>

```yaml
handshake: '{{if eq .event.type "url_verification"}}{{.event.challenge}}{{end}}'   # Slack Events API
handshake: '{{index .query "hub.challenge"}}'                                       # a GET check, such as Meta
verify:                                                                             # Stripe
  secret: ["secret:stripe_webhook_secret"]
  message: '{{kv (index .headers "Stripe-Signature") "t"}}.{{.body}}'
  signature: '{{kv (index .headers "Stripe-Signature") "v1"}}'
```

</div>

kdeps saves the read position of a `file` listener and the cursor of a `poll` listener under `~/.kdeps/projects/`, never in the project. A restart goes on after the last event that ran. Each finished run prints `[listen ID] RESULT`; a failed run prints the error, and the listener goes on.

## Use cases

- Serve a workflow as an HTTP JSON API for a frontend or another service
  (`apiServer:` under `deploy:`).
- Run a workflow as a chat bot on Discord, Slack or Telegram
  (`listen:` with `discord_gateway`, `slack_events` or `telegram_updates`).
- Process one file per invocation from a pipeline, cron job, or CI step
  (`file`).
- Run a workflow for each new log line, queued job, API item or webhook
  (`listen:`).

## See also

- [Workflow configuration](/workflow/configuration) - full `deploy` reference
- [Workflow](/workflow/) - the request and the response
- [Connectors](/workflow/resources/connectors) - send the reply with `slack_post`, `telegram_post`, `discord_post`
