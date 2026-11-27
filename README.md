# kdeps

[![Build and Test](https://github.com/kdeps/kdeps/actions/workflows/build-test.yml/badge.svg?branch=main)](https://github.com/kdeps/kdeps/actions/workflows/build-test.yml)
[![Coverage](https://codecov.io/gh/kdeps/kdeps/branch/main/graph/badge.svg)](https://codecov.io/gh/kdeps/kdeps)
[![Release](https://img.shields.io/github/v/tag/kdeps/kdeps?sort=semver&label=release)](https://github.com/kdeps/kdeps/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/kdeps/kdeps)](https://go.dev/)
[![License](https://img.shields.io/github/license/kdeps/kdeps)](https://github.com/kdeps/kdeps/blob/main/LICENSE)
[![CodeQL](https://github.com/kdeps/kdeps/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/kdeps/kdeps/actions/workflows/codeql.yml)
[![Docs](https://github.com/kdeps/kdeps/actions/workflows/docs.yml/badge.svg?branch=main)](https://kdeps.com)
[![Documentation](https://img.shields.io/badge/docs-kdeps.com-00E5FF)](https://kdeps.com)
[![Registry](https://img.shields.io/badge/registry-kdeps.io-00E5FF)](https://kdeps.io)
[![GitHub stars](https://img.shields.io/github/stars/kdeps/kdeps)](https://github.com/kdeps/kdeps/stargazers)

kdeps turns one `workflow.yaml` into an offline AI appliance. An API call, a chat message or a webhook starts one bounded run.

1. **Workflows** - a YAML pipeline. Each event starts one run: an API call, a file, a prompt, or a listener (Telegram, Slack, Discord, WhatsApp, a webhook, a websocket, a polled API). Connectors send the answer.
2. **Agents** - a `chat` with `loop:` calls tools, as one step of the workflow.
3. **Deploy** - ship the same YAML as an image, a binary, or an ISO, on your own infrastructure.

## Install

```bash
brew install kdeps/tap/kdeps                                              # macOS or Linux
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh   # or the install script
```

```powershell
irm https://raw.githubusercontent.com/kdeps/kdeps/main/install.ps1 | iex   # Windows PowerShell
```

See the [installation guide](https://kdeps.com/start/installation) for other options.

## 1. Workflows

A workflow is a YAML pipeline. Its resources run in `requires:` order, and a check can return 400 before any model call.

```yaml
# resources/llm.yaml
actionId: llm
validations:
  check:
    - get('q') != ''           # 400 before any model call
chat:
  model:
    name: llama3.2:1b          # local model, no API key
  context:
    prompt: "{{ get('q') }}"
```

```bash
kdeps new my-agent && cd my-agent
export KDEPS_API_AUTH_TOKEN=dev-token   # local testing only
kdeps run workflow.yaml                 # serve POST /api/v1/chat on :16395
```

`kdeps test .` runs the `evals:` of the workflow.

A listener starts one run for each event, and a connector sends the answer. This is a Telegram bot:

```yaml
# workflow.yaml
deploy:
  input:
    listen:
      - listener: telegram_updates   # one run per message; token from the vault
```

```yaml
# resources/reply.yaml
actionId: reply
flow:
  requires: [llm]
component:
  name: telegram_post                # connector: one Bot API call
  with:
    path: sendMessage
    body:
      chat_id: "{{ input('message').chat.id }}"
      text: "{{ get('llm').message.content }}"
```

See the [quickstart](https://kdeps.com/workflow/quickstart), [listeners](https://kdeps.com/workflow/input-sources), [connectors](https://kdeps.com/workflow/resources/connectors) and the [examples](https://kdeps.com/examples/).

## 2. Agents

A `chat` with `loop:` calls tools over many rounds. It is one step of the workflow.

```yaml
# resources/assistant.yaml
chat:
  loop:
    tools: [read_file, edit_file, bash_exec]   # the tools the agent may call
```

```bash
kdeps new helper --template assistant && cd helper && kdeps
```

See the [agent docs](https://kdeps.com/workflow/agent-loop/).

## 3. Deploy on your infrastructure

Ship the same YAML as an image, Kubernetes manifests, an ISO, or one binary.

```bash
kdeps bundle build .                                          # container image
kdeps bundle prepackage my-agent-1.0.0.kdeps --include-models # binary with the model
```

API keys go in an encrypted vault (`kdeps secret set NAME`), read with `secret('NAME')` and by connectors such as `github_get` and `slack_post`. The vault ships with the image; give it `KDEPS_VAULT_PASSWORD` at run time.

See [deploy](https://kdeps.com/deploy/), [secrets](https://kdeps.com/workflow/secrets) and [LLM server](https://kdeps.com/llm-server/).

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md). kdeps has the [Apache 2.0](LICENSE) license.
