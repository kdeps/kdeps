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

kdeps builds git-native AI appliances: your AI agent is YAML in your repository, built once and run where you need it.

1. **Workflows** - deterministic YAML pipelines served as an HTTP API, bot or file processor.
2. **Agent** - `kdeps` opens an AI REPL with tools and memory, on a local model or an LLM server you run.
3. **Deploy on your infrastructure** - the same YAML ships as a Docker image, Kubernetes manifests, a bootable ISO or one binary.

## Install

```bash
brew install kdeps/tap/kdeps                                              # macOS or Linux
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh   # or the install script
```

```powershell
irm https://raw.githubusercontent.com/kdeps/kdeps/main/install.ps1 | iex   # Windows PowerShell
```

Other options: [installation guide](https://kdeps.com/start/installation).

## 1. Workflows

Save as `workflow.yaml`:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: response      # the resource whose output is returned
settings:
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
      model: llama3.2:1b        # local model, no API key
      prompt: "{{ get('q') }}"
      timeout: 60s
  - actionId: response
    name: Return the answer
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

Resources run in dependency order, independent ones in parallel, on the same scheduler as agent mode: the model's wording varies, the path does not. A chat's `judges:` (or a preset's, with `judgePreset:`) review and revise its answer, as agent mode's judge panel does. `harness: [accuracy, honesty]` adds agent mode's system-prompt sections. `refine: true` rewrites a terse prompt first, as `/refine` does. A chat lists agent mode's tools by name (`web_search`, `calculator`, `subagent`, ...). `kdeps test .` runs a workflow's `tests:` in-process, with no port bound.

More: [workflow docs](https://kdeps.com/workflow/), [examples](https://kdeps.com/examples/), [agencies](https://kdeps.com/agencies/) (several agents as one system).

## 2. Agent

```bash
kdeps                            # REPL with built-in tools (files, shell, search, memory)
kdeps ./my-agent/ --model llama3.2   # also load that project's workflows as tools
```

The model picks which tools to call. File tools ask before touching a path outside the working directory, except `~/.kdeps` and `/tmp`.

More: [agent docs](https://kdeps.com/agent/), [desktop app](https://kdeps.com/agent/desktop#install).

## 3. Deploy on your infrastructure

```bash
kdeps bundle build .        # Docker image
kdeps export k8s            # Kubernetes manifests
kdeps export iso            # bootable ISO
kdeps bundle prepackage     # standalone binary
```

Point agents at an LLM server you host, so prompts and data stay on your network:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1   # your OpenAI-compatible server
  models:
    - llama3.2
```

`kdeps llm wizard` builds that server. More: [deploy](https://kdeps.com/deploy/), [LLM server](https://kdeps.com/llm-server/), [security](https://kdeps.com/reference/security).

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md). Licensed under [Apache 2.0](LICENSE).
