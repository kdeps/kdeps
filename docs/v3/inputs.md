# Inputs (API, bot, file)

Workflows declare how work arrives under `settings.input`.

```yaml
settings:
  input:
    sources: [api]           # default — HTTP
    # sources: [bot]
    # sources: [file]
    # sources: [api, bot]    # combine
```

Omit `settings.input` → API server (default).

## API

`settings.apiServer` routes + auth. Terminal resource uses `apiResponse:`. See [Workflow](/workflow) · [Security](/security).

## Bot

Platforms: Discord, Slack, Telegram, WhatsApp.

**Secrets** in `~/.kdeps/config.yaml` (or env). Workflow only enables platforms:

```yaml
# ~/.kdeps/config.yaml
bot_connections:
  telegram:
    botToken: "${TELEGRAM_BOT_TOKEN}"
  slack:
    botToken: "${SLACK_BOT_TOKEN}"
    appToken: "${SLACK_APP_TOKEN}"
    signingSecret: "${SLACK_SIGNING_SECRET}"

# workflow.yaml
settings:
  input:
    sources: [bot]
    bot:
      executionType: poll          # or stateless
      telegram:
        pollIntervalSeconds: 1
```

| `executionType` | Behavior |
|-----------------|----------|
| `poll` (default) | Long-running; platform connection |
| `stateless` | One JSON message on stdin → reply stdout → exit (FaaS / tests) |

Read message:

```yaml
before:
  - set('text', get('message').text)
  - set('platform', get('message').platform)
```

Reply with **`botReply:`** (not `apiResponse:`):

```yaml
botReply:
  text: "{{ get('generateReply') }}"
```

| Source | Terminal action |
|--------|-----------------|
| `api` | `apiResponse:` |
| `bot` | `botReply:` |

Combined sources: include both response types as needed.

## File

Single-shot file or stdin processing:

```yaml
settings:
  input:
    sources: [file]
```

```bash
kdeps run workflow.yaml --file ./doc.txt
# or pipe / KDEPS_FILE_PATH
```

`--file` wins over stdin and config. See `kdeps run --help`.

## Examples in repo

`examples/telegram-bot/`, `examples/stateless-bot/`, `examples/file-processor/`.

[Workflow](/workflow) · [Resources](/resources) · [CLI](/cli).
