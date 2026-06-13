# Examples

Complete, runnable workflows that demonstrate common patterns. Every example is copy-paste ready -- clone, add your API keys, and run.

## Overview

Each example targets a different use case. Pick the one closest to what you're building:

| Example | Mode | What it demonstrates |
|---|---|---|
| [Stateless Bot](/examples/stateless-bot/) | Workflow | One-shot stdin/stdout LLM calls -- cron jobs, CI pipelines |
| [Telegram Bot](/examples/telegram-bot/) | Workflow | Polling loop, multi-resource pipelines, external API calls |
| [Showcase](/examples/showcase) | Workflow | Complex agents in ~20 lines of YAML -- multiple real-world patterns |

## Stateless Bot

A one-shot bot that reads from stdin (or an env var), calls an LLM, and writes the reply to stdout. No server, no polling, no state.

Best for:
- Cron jobs that summarize data
- CI pipeline steps that classify or label
- Custom integrations that call kdeps as a subprocess

```bash
echo "What is 2+2?" | kdeps run workflow.yaml
```

## Telegram Bot

A polling bot that watches for Telegram messages and replies with LLM responses. Two resources chained together: `llm` calls the model, `reply` sends the answer back via the Telegram API.

Best for:
- Chatbot interfaces over existing workflow resources
- Notification-driven pipelines
- Multi-resource orchestration patterns

```bash
KDEPS_TELEGRAM_BOT_TOKEN=... kdeps run workflow.yaml
```

## Showcase

A collection of real-world agents -- each a complete workflow you can POST to and get structured JSON back. Covers data extraction, classification, summarization, and more.

Best for:
- Seeing how complex agents fit in ~20 lines of YAML
- Learning the `POST /api/v1/run` pattern
- Adapting a pattern to your own data

## See Also

- [Quickstart](/getting-started/quickstart) -- build your first workflow in 5 minutes
- [Workflow Mode](/modes/workflow-mode) -- deterministic DAG execution
- [Agent Mode](/modes/agent-loop-mode) -- interactive LLM-driven tool calling
