# Bot reply resource

The `botReply:` resource sends a text reply back to the bot platform that delivered the current message - Discord, Slack, Telegram, WhatsApp, or stdout when running in stateless mode. It only makes sense in a workflow whose `settings.input.sources` includes `bot`.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). Requires `bot` configured under `settings.input` - see [Input Sources](/concepts/input-sources).

## Basic usage

```yaml
# resources/llm.yaml
actionId: llm
name: Generate Reply
chat:
  model: llama3.2:1b
  prompt: "{{ get('message') }}"
```

```yaml
# resources/reply.yaml
actionId: reply
name: Send Reply
requires: [llm]
botReply:
  text: "{{ get('llm').message.content }}"
```

## Configuration options

| Option | Description |
|---|---|
| `text` | The message to send. Supports `{{ }}` expression interpolation (required) |

## Output

```json
{ "success": true }
```

## Full example

See the [Stateless Bot](/examples/stateless-bot/) and [Telegram Bot](/examples/telegram-bot/) example agents for complete, runnable `botReply:` workflows.

## See also

- [Input sources](/concepts/input-sources) - configuring `settings.input.bot`
- [Resources overview](overview) - all resource types
