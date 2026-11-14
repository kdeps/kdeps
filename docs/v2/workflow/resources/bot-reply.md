# Bot replies

A bot workflow replies with its result: kdeps sends the workflow's result back to the chat that delivered the message - Discord, Slack, Telegram, or stdout in stateless mode. There is no separate reply resource. End the workflow with an `apiResponse:` (or any resource whose output is text) and set it as `targetActionId`.

This is workflow mode with `bot` in `sources:` under `settings:` `input:` - see [Input sources](/workflow/input-sources).

```text
message -> workflow (llm -> reply) -> result text -> same chat
```

## Basic usage

<div v-pre>

```yaml
# workflow.yaml (abridged)
metadata:
  targetActionId: reply        # its result is the reply
deploy:
  input:
    sources: [bot]
    bot:
      executionType: polling
      telegram: {}   # presence enables the platform; credentials are in ~/.kdeps/config.yaml bot_connections
```

```yaml
# resources/llm.yaml
actionId: llm
name: Generate Reply
chat:
  model:
    name: llama3.2:1b
  context:
    prompt: "{{ input('message') }}"
```

```yaml
# resources/reply.yaml
actionId: reply
name: Reply
flow:
  requires: [llm]
apiResponse:
  success: true
  response: "{{ get('llm').message.content }}"   # the text sent to the chat
```

</div>

## What is sent

| Workflow result | Reply text |
|---|---|
| `apiResponse` with a string `response` | That string |
| `apiResponse` with a map or list `response` | The response as JSON |
| A `chat` result (target is the chat resource itself) | `message.content` |
| A plain string | The string |
| Anything else | The result as JSON |

An empty result sends nothing. A failed workflow sends nothing, and kdeps logs it. If users must get a reply, send a fallback with the `onError:` of the resource.

## Stateless mode

`executionType: stateless` reads one message as JSON on stdin and prints the reply on stdout:

```bash
echo '{"message":"What is 2+2?","platform":"cli"}' | kdeps run workflow.yaml
# 4
```

## Full example

See the [Stateless bot](/examples/stateless-bot/) and [Telegram bot](/examples/telegram-bot/) examples.

## See also

- [Input sources](/workflow/input-sources) - configuring `bot:` under `settings:` `input:`
- [API response](/workflow/resources/api-response) - the `apiResponse:` block
