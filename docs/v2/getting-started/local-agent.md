---
title: Run Locally in 30 Seconds
description: Install kdeps and start an AI agent REPL on your machine. No Docker, no config, no API key required.
---

# Run Locally in 30 Seconds

kdeps ships as a standalone binary. Install it, run it, and you have an interactive AI agent running on your machine. No Docker. No config file. No API key required if you use a local model.

## Install

**macOS (Homebrew):**

```bash
brew install kdeps/tap/kdeps
```

**macOS / Linux (curl):**

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

**Go install:**

```bash
go install github.com/kdeps/kdeps@latest
```

Verify:

```bash
kdeps --version
```

## Start the agent

```bash
kdeps
```

That's it. kdeps opens an interactive REPL. By default it downloads and runs a local llamafile model (~1.1 GB on first run, cached in `~/.kdeps/models/`). No API key needed.

```text
kdeps v2.x.x
model: llama3.2:1b (local, file backend)
> _
```

Type anything and the agent responds. The model runs entirely on your machine.

## Use a local model via Ollama

If you already have [Ollama](https://ollama.com) installed, point kdeps at it:

```bash
# pull a model with Ollama
ollama pull llama3.2

# start kdeps using that model
kdeps --model llama3.2 --backend ollama
```

Or try a reasoning model:

```bash
ollama pull deepseek-r1
kdeps --model deepseek-r1 --backend ollama
```

Your prompts never leave the machine.

## Use a cloud model

Set your API key and pick a provider:

```bash
# Anthropic
export ANTHROPIC_API_KEY=sk-ant-...
kdeps --model claude-opus-4-5 --backend anthropic

# OpenAI
export OPENAI_API_KEY=sk-...
kdeps --model gpt-4o --backend openai
```

The `--backend` flag tells kdeps where to route the request. The `--model` flag picks the model. Both can also be set in `~/.kdeps/config.yaml` to avoid repeating them on every invocation.

## Load your own workflows as tools

Once you have a workflow directory, pass it to `kdeps serve`:

```bash
kdeps serve ./my-workflow/
```

The REPL starts with your workflow registered as a callable tool. The LLM decides when to invoke it. This is [agent mode](/modes/agent-loop-mode) - the LLM drives, your workflows execute on demand.

## What to do next

- [Local Models (Llamafile & Ollama)](/getting-started/local-models) - Go deeper on offline setup, model selection, and privacy
- [Quickstart](/getting-started/quickstart) - Build your first workflow
- [Agent Skills](/getting-started/agent-skills) - Extend the agent with pre-built skill sets
