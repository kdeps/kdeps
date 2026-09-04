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

> Before moving on, please consider giving us a GitHub star ⭐️. Thank you!

**AI Appliance Builder** - YAML-defined AI agents, workflows, and agencies.

kdeps works at three levels:

- **Local agent** - run `kdeps` and you are in an autonomous AI REPL: tool use, memory, fully offline against a local model. No config, no API key.
- **Workflow** - define what the agent does in one `workflow.yaml` and run it as an HTTP API, a bot, or a file processor. Same file, laptop or server.
- **Appliance** - ship that file unchanged as a Docker image, Kubernetes manifests, a bootable ISO, or a single binary.

One YAML file replaces a Python script wiring together an LLM SDK, a web server, retry logic, and a Dockerfile.

## Quickstart

```yaml
# workflow.yaml - a two-resource chat API. No LLM server needed: the default
# model (llama3.2:1b) runs as a local llamafile, downloaded on first run.
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: response   # the resource whose output is the HTTP response
settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
resources:
  - actionId: llm
    name: LLM Chat
    validations:
      methods: [POST]
      routes: [/api/v1/chat]
      check:
        - get('q') != ''      # 'q' comes from the JSON body
      error:
        code: 400
        message: "'q' is required"
    chat:
      model: llama3.2:1b      # or an ollama/openai/anthropic/groq model
      role: user
      prompt: "{{ get('q') }}"
      timeout: 60s
  - actionId: response
    name: API Response
    requires: [llm]            # runs after 'llm' - the DAG resolves order
    apiResponse:
      success: true
      response:
        answer: get('llm').message.content
```

```bash
# KDEPS_API_AUTH_TOKEN is the HTTP endpoint's bearer token - not an LLM key
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run workflow.yaml

curl -s -X POST localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q": "What is entropy, in one sentence?"}'
# {"success": true, "data": {"answer": "Entropy is a measure of disorder..."}}
```

More examples: [kdeps.com/examples](https://kdeps.com/examples/) (22 walkthroughs, each a working project).

## Install

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/kdeps/kdeps/main/install.ps1 | iex
```

Or with Homebrew (macOS and Linux):

```bash
brew install kdeps/tap/kdeps
```

## How it works

Whichever level you use, a workflow runs in one of two execution modes - and an agency bundles several workflows into one system.

**Workflow mode** - DAG-deterministic request/response pipelines. Each resource declares its dependencies via `requires:` and runs in order, ending in an `apiResponse`.

The LLM call inside a `chat:` resource is still probabilistic, but the pipeline around it is not: the same request takes the same path, validation runs before any model call, and the response is shaped to a fixed schema.

Resources cover LLM chat, HTTP, Python, shell, SQL, email, web scraping, browser automation, embeddings, local and web search, and calls to other agents or components. Expressions (`get()`, `output()`, `set()`, plus Jinja2 control flow) wire the steps together.

```bash
kdeps run workflow.yaml          # local, instant startup
kdeps run ./my-agent/            # or point at a directory containing workflow.yaml
kdeps run workflow.yaml --dev    # hot reload
```

**Agent loop mode** - an autonomous LLM loop. Every workflow becomes a callable tool, and the LLM decides which to call, in what order, to complete the task. Runs as an interactive REPL until you exit (Ctrl+D).

```bash
kdeps                            # bare agent loop REPL
kdeps ./my-agent/                # register the workflow as an LLM-callable tool
kdeps ./my-agent/ --model llama3.2 --system "You are a DevOps assistant."
kdeps --stealth                  # "Muted" UI: dark gray, model name barely visible (for use in public)
```

**Agencies** - a collection of agents that work together. Each agent is its own `workflow.yaml` with its own resources, model, and logic, wired together with the `agent:` resource type - like calling a function, but the function is an entire AI pipeline.

```bash
kdeps run agency.yaml
```

Docs: [Workflow mode](https://kdeps.com/modes/workflow-mode) · [Agent loop mode](https://kdeps.com/modes/agent-loop-mode) · [Agencies](https://kdeps.com/concepts/agency) · [Resources overview](https://kdeps.com/resources/overview)

## Build and deploy

The workflow you run locally exports unchanged to any target:

```bash
kdeps bundle build .        # Docker image
kdeps export iso            # bootable edge ISO
kdeps bundle prepackage     # self-contained binary per arch
kdeps export k8s            # Kubernetes manifests
```

For a public HTTPS endpoint, add static PEM files or automatic Let's Encrypt to `workflow.yaml`, point DNS at the host, and open ports 80 and 443:

```yaml
settings:
  letsEncrypt:
    domain: api.example.com
    email: ops@example.com
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 443
```

Docs: [Deployment guide](https://kdeps.com/guides/deployment-guide) · [TLS and HTTPS](https://kdeps.com/deployment/tls-https)

## Reference

- **Registry** - `kdeps registry search|install|submit` for pre-built components. [kdeps.io](https://kdeps.io)
- **Agent skill** - `npx skills add https://github.com/kdeps/skill --skill kdeps` teaches Claude Code, Cursor, and other agents to scaffold kdeps projects. [Docs](https://kdeps.com/getting-started/agent-skills)
- **LLM server appliance** - `kdeps llm wizard` builds a standalone OpenAI-compatible inference server (`ollama`, `llamafile`, `gguf`, `vllm`, `tgi`, `sglang`, and more), no workflow path required. [Docs](https://kdeps.com/deployment/llm-server) · [Commands](https://kdeps.com/reference/cli/llm)
- **Global config** - machine-local settings (LLM backend, API keys, SQL/SMTP/IMAP connections) live in `~/.kdeps/config.yaml`, never in `workflow.yaml`. `kdeps edit` to open it, `kdeps doctor` to check it. [Docs](https://kdeps.com/configuration/advanced)
- **Security** - when `apiServer` is set, requests require a bearer token (`KDEPS_API_AUTH_TOKEN`) and pass through rate-limit, body-size, and concurrency caps before reaching the DAG. [Docs](https://kdeps.com/reference/security)
- **Logging** - structured JSON via `log/slog`. `KDEPS_LOG_FORMAT=json` for production; default level WARN; `--verbose` (INFO), `--debug` (DEBUG).
- **Book** - [*AI Appliances - Build & Deploy Autonomous AI Agents and Agencies in YAML*](https://leanpub.com/kdeps). Free (PDF, EPUB, web).

---

[Documentation](https://kdeps.com) | [Registry](https://kdeps.io) | Apache 2.0
