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

# What kdeps is overall:
- kdeps (short for **"Knowledge Dependencies"**) formalizes AI workflows not as fragile ad-hoc scripts, but as a Directed Acyclic Graph (DAG) of typed knowledge and compute dependencies. It enforces data contracts between prompts, tools, and code at runtime—if required upstream data is missing, execution is halted before costly LLM calls.
- **Local-First & Offline-First Philosophy:** Engineered for organizations that require zero-cloud-dependency AI. Eliminates vendor lock-in, recurring per-token fees, and external API latency. Operates fully air-gapped using local LLMs (Ollama, local GGUF/llamafile) and local embeddings, satisfying strict GDPR, HIPAA, and enterprise data-residency compliance.

kdeps turns one workflow.yaml into an offline API, a binary, an image, or an ISO. You review that file as a pull request. kdeps test runs its evals.

1. **Workflows** - a YAML pipeline is the one way to run kdeps. Feed it from an HTTP API, a bot, a file, or an interactive prompt.
2. **Agents** - a `chat` with `loop:` is an AI agent with tools and memory, inside the same workflow. `kdeps new helper --template assistant` starts one.
3. **Deploy on your infrastructure** - ship the same YAML as a Docker image, Kubernetes manifests, a bootable ISO, or one binary.

## Install

```bash
brew install kdeps/tap/kdeps                                              # macOS or Linux
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh   # or the install script
```

```powershell
irm https://raw.githubusercontent.com/kdeps/kdeps/main/install.ps1 | iex   # Windows PowerShell
```

For other options, see the [installation guide](https://kdeps.com/start/installation). If you upgrade from v2, see [what changed](https://kdeps.com/start/installation#upgrading-from-v2).

## 1. Workflows

Save as `workflow.yaml`:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: response      # the resource whose output is returned
deploy:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
resources:
  - actionId: llm
    name: Ask the AI
    validations:
      check:
        - get('q') != ''        # reject the request before any model call
      error:
        code: 400
        message: "'q' is required"
    chat:
      model:
        name: llama3.2:1b        # local model, no API key
        timeout: 60s
      context:
        prompt: "{{ get('q') }}"
  - actionId: response
    name: Return the answer
    flow:
      requires: [llm]             # runs after llm
    apiResponse:
      success: true
      response:
        answer: get('llm').message.content
```

Run it and ask a question:

```bash
export KDEPS_API_AUTH_TOKEN=dev-token   # local testing only
kdeps run workflow.yaml

curl -s -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q": "Explain entropy in one sentence."}'
```

Resources run in dependency order. Independent resources run in parallel. The model can write a different answer each time. But for the same request, kdeps always runs the same steps in the same order.

A chat puts its keys in these groups: `model`, `context`, `harness`, `tools`, `rag`, and `guardrails`. Guardrail judges review and revise the answer. `tools.list` names built-in tools (`web_search`, `calculator`, `subagent`, ...), or `auto` for all of them. Judges (`guardrails.judges: auto`) are opt-in. They make a small local model answer better. `harness.promptMode` sends short harness text after the first request. A workflow ships its own limits and prompts in `events/` and `harness/` folders; every package and image carries them. See [Answer quality](https://kdeps.com/workflow/quality).

`kdeps test .` runs the `evals:` of a workflow in the process. It runs assertions, judges, and repeated runs against a pass-rate threshold.

For more data, see the [workflow docs](https://kdeps.com/workflow/), [examples](https://kdeps.com/examples/), and [agencies](https://kdeps.com/agencies/) (many agents as one system).

## 2. Agents

An agent is a workflow whose `chat` has a `loop:` block: the model calls tools over many rounds and keeps one conversation per `loop.session` id.

```bash
kdeps new helper --template assistant   # interactive prompt + chat with loop:
cd helper && kdeps                      # bare kdeps runs ./workflow.yaml
```

```yaml
# helper/resources/assistant.yaml
chat:
  context:
    prompt: "{{ input('message') }}"
  loop:
    tools: [read_file, edit_file, bash_exec]   # the tools the agent may call
    session: "{{ get('session') }}"           # one conversation across prompts
```

File tools stay inside the working directory. `~/.kdeps` and `/tmp` are exceptions.

For more data, see the [agent docs](https://kdeps.com/workflow/agent-loop/).

## 3. Deploy on your infrastructure

```bash
kdeps bundle build .                                          # container image
kdeps export k8s .                                            # Kubernetes manifests
kdeps export iso .                                            # bootable ISO
kdeps bundle package .                                        # my-agent-1.0.0.kdeps
kdeps bundle prepackage my-agent-1.0.0.kdeps --include-models # binary with the model
```

Connect agents to an LLM server that you host. Then prompts and data stay on your network:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1   # your OpenAI-compatible server
  models:
    - llama3.2
```

`kdeps llm run` builds and runs that server. For more data, see [deploy](https://kdeps.com/deploy/), [LLM server](https://kdeps.com/llm-server/), and [security](https://kdeps.com/reference/security).

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md). kdeps has the [Apache 2.0](LICENSE) license.
