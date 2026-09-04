---
title: Run locally in 30 seconds
description: Install kdeps and start an AI agent REPL on your machine. No Docker, no config, no API key required.
---

# Run locally in 30 seconds

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
go install github.com/kdeps/kdeps/v2@latest
```

Verify:

```bash
kdeps --version
```

## Start the agent

```bash
kdeps
```

That's it. kdeps opens an interactive REPL. By default it uses the local llamafile model `llama3.2:1b` (~1.1 GB). On first use you confirm the download; it is cached in `~/.kdeps/models/`. No API key needed.

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

Once you have a workflow directory, pass it as an argument to `kdeps`:

```bash
kdeps ./my-workflow/
```

The REPL starts with your workflow registered as a callable tool. The LLM decides when to invoke it. This is [agent mode](/modes/agent-loop-mode) - the LLM drives, your workflows execute on demand.

## Next steps

- [Local models (llamafile and Ollama)](/getting-started/local-models) - offline setup, model selection, privacy
- [Quickstart](/getting-started/quickstart) - build your first workflow
- [Agent skills](/getting-started/agent-skills) - extend the agent with pre-built skill sets
