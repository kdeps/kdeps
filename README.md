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

Build and deploy AI agents in YAML. Two modes: **workflow** (DAG pipelines), **agent** (autonomous LLM loop). Git-native: everything lives in versionable YAML you commit to your repo like any other code.

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

## Book

[<img src="https://d2sofvawe08yqg.cloudfront.net/kdeps/s_hero?1779817160" alt="AI Appliances book cover" width="140" align="right" style="margin-left:16px">](https://leanpub.com/kdeps)

**[AI Appliances - Build & Deploy Autonomous AI Agents and Agencies in YAML](https://leanpub.com/kdeps)**
Free. PDF, EPUB, and web.

Hands-on guide covering deterministic pipelines, multi-agent orchestration, error handling, and vendor-agnostic deployment - the production challenges most AI frameworks leave to you.

<br clear="right">

## Modes

### Workflow mode

DAG-deterministic request/response pipelines: each resource declares its dependencies via `requires:` and runs in order, ending in an `apiResponse`. Resource types cover `chat`, `httpClient`, `python`, `exec`, `sql`, `email`, `scraper`, `browser`, `embedding`, `searchLocal`, `searchWeb`, `agent`, and `component`; expressions (`get()`, `output()`, `set()`, plus Jinja2 control flow) wire steps together.

```bash
kdeps run workflow.yaml          # local, instant startup
kdeps run ./my-agent/            # or point at a directory containing workflow.yaml
kdeps run workflow.yaml --dev    # hot reload
```

Docs: [Workflow Mode](https://kdeps.com/modes/workflow-mode) · [Resources Overview](https://kdeps.com/resources/overview) · [Expressions](https://kdeps.com/concepts/expressions)

### Agent Loop mode

Autonomous LLM loop: every workflow becomes a callable tool, and the LLM decides which to call, in what order, to complete the task. Runs as an interactive REPL until you exit (Ctrl+D).

```bash
kdeps # agent loop REPL

# Advanced usage: point at a workflow or agency directory to register it as a tool
kdeps ./my-agent/ # registers the workflow as an LLM-callable tool
kdeps ./my-agent/ --model llama3.2 --system "You are a DevOps assistant." # override model/system prompt
```

```
KDEPS_AGENT_MODEL=deepseek-v4-flash   # override model via env
KDEPS_AGENT_BACKEND=deepseek
```

Docs: [Agent Loop Mode](https://kdeps.com/modes/agent-loop-mode)

## Agencies

An agency is a collection of agents that work together. Each agent is its own `workflow.yaml` with its own resources, model, and logic, wired together with the `agent:` resource type — like calling a function, but the function is an entire AI pipeline.

```bash
kdeps run agency.yaml
```

Docs: [Agencies](https://kdeps.com/concepts/agency) · Example: [`examples/agency/`](examples/agency/)

## Build and deploy

```bash
kdeps bundle build          # Docker image
kdeps bundle export iso     # bootable edge ISO
kdeps bundle prepackage     # self-contained binary per arch
kdeps export k8s            # Kubernetes manifests
```

### HTTPS (custom domain)

Static PEM files or automatic **Let's Encrypt** for your domain:

```yaml
# workflow.yaml
settings:
  letsEncrypt:
    domain: api.example.com
    email: ops@example.com
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 443
```

Point DNS at the host; open ports **80** and **443**.

Docs: [TLS and HTTPS](https://kdeps.com/deployment/tls-https)

### LLM server appliance

Standalone OpenAI-compatible inference server (no workflow path). Stock engines: `ollama`, `llamafile`, `llama-server` / `gguf`, `llamacpp`, `vllm`, `tgi`, `sglang`, `localai`, `openai-compat`.

```bash
kdeps llm wizard   # TUI: engine + harvest model + build
kdeps llm models   # harvest (llamafile + GGUF) available
kdeps llm list
kdeps llm show vllm
kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
kdeps llm build --engine vllm --model facebook/opt-125m --gpu cuda --tag myorg/vllm:1
kdeps llm export k8s --image myorg/llm:1 --engine ollama -o llm.yaml
kdeps llm client-config --url http://192.168.1.50:8000/v1
```

Point any kdeps host at the appliance:

```yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1
```

Docs: [LLM Server Appliance](https://kdeps.com/deployment/llm-server) · [LLM Commands](https://kdeps.com/reference/cli/llm) · Example: [`examples/llm-server/`](examples/llm-server/)

## Registry

```bash
kdeps registry search <query>
kdeps registry install <package>
kdeps registry submit --tag v1.0.0   # generate formula for kdeps.io PR
```

## Agent skill

A [coding-agent skill](https://github.com/kdeps/skill) teaches Claude Code, Cursor,
Grok, and other agents how to scaffold kdeps workflows, components, and agencies —
including `kdeps.pkg.yaml` for [kdeps.io](https://kdeps.io) distribution.

```bash
npx skills add https://github.com/kdeps/skill --skill kdeps
```

Docs: [Agent Skills](https://kdeps.com/getting-started/agent-skills)

## Global config

Machine-local settings — LLM backend, API keys, SQL/SMTP/IMAP connections, per-resource-type defaults — live in `~/.kdeps/config.yaml`, never in `workflow.yaml`. Override any of it per-agent with an `agents:` block keyed by the workflow name.

```bash
kdeps edit    # opens ~/.kdeps/config.yaml
kdeps doctor  # check config, Ollama, Python, installed agents
```

Docs: [Global Config Reference](https://kdeps.com/configuration/advanced) · [LLM Providers](https://kdeps.com/reference/llm-providers)

## Security

When `apiServer` is configured, authentication is required (`Authorization: Bearer <token>` or `X-Api-Key: <token>`, token set via `KDEPS_API_AUTH_TOKEN` or `~/.kdeps/config.yaml`, never in `workflow.yaml`). Every request also passes through rate limiting, a body-size cap, and a concurrency cap before reaching the workflow DAG.

```bash
export KDEPS_API_AUTH_TOKEN=your-secret-token
kdeps run workflow.yaml
```

Docs: [Security Reference](https://kdeps.com/reference/security)

## Logging

Structured JSON via `log/slog`. Set `KDEPS_LOG_FORMAT=json` for production output. Default level: WARN. Flags: `--verbose` (INFO), `--debug` (DEBUG).

---

[Documentation](https://kdeps.com) | [Registry](https://kdeps.io) | Apache 2.0
