# Examples

Complete, runnable workflows that demonstrate common patterns. Every example is copy-paste ready - clone, add your API keys, and run.

## Overview

Each example targets a different use case. Pick the one closest to what you're building:

| Example | Mode | What it demonstrates |
|---|---|---|
| [Document summarizer](/examples/file-processor) | Workflow | The `file` input source - read one file, return JSON, exit |
| [Batch processing](/examples/batch-processing) | Workflow | `items:` iteration - process a list in one request |
| [Document search (RAG)](/examples/rag-search) | Workflow | `embedding:` upsert and search, two routes in one workflow |
| [Stateless bot](/examples/stateless-bot/) | Workflow | One-shot stdin/stdout LLM calls - cron jobs, CI pipelines |
| [Telegram bot](/examples/telegram-bot/) | Workflow | Polling loop, multi-resource pipelines, external API calls |
| [Showcase](/examples/showcase) | Workflow | Complex agents in ~20 lines of YAML - multiple real-world patterns |

## Document summarizer

A single-shot workflow that reads a document from `--file`, stdin, or an
environment variable, sends it to a local LLM, and prints a structured JSON
summary. Runs once and exits.

Best for:
- Cron jobs and CI steps
- `kdeps run ... | jq` one-liners
- Any pipeline that treats kdeps as a subprocess

[Build it step by step](/examples/file-processor)

## Batch processing

An API that takes a list of items in one request, fans out an HTTP call per
item with `items:`, transforms each result, and returns an aggregated summary.

Best for:
- Enriching a list of records from an external API
- Running the same LLM prompt over many inputs
- Any fan-out / collect pattern

[Build it step by step](/examples/batch-processing)

## Document search (RAG)

A two-route API: `POST /index` stores a document, `POST /search` returns the
closest matches. Uses the built-in `embedding:` resource - a local SQLite
index, no vector database.

Best for:
- The retrieval half of a RAG pipeline
- A lightweight search endpoint over your own text
- Learning `validations.routes` scoping

[Build it step by step](/examples/rag-search)

## Stateless bot

A one-shot bot that reads from stdin (or an env var), calls an LLM, and writes the reply to stdout. No server, no polling, no state.

Best for:
- Cron jobs that summarize data
- CI pipeline steps that classify or label
- Custom integrations that call kdeps as a subprocess

```bash
echo "What is 2+2?" | kdeps run workflow.yaml
```

## Telegram bot

A polling bot that watches for Telegram messages and replies with LLM responses. Two resources chained together: `llm` calls the model, `reply` sends the answer back via the Telegram API.

Best for:
- Chatbot interfaces over existing workflow resources
- Notification-driven pipelines
- Multi-resource orchestration patterns

```bash
TELEGRAM_BOT_TOKEN=... kdeps run workflow.yaml
```

## Showcase

A collection of real-world agents - each a complete workflow you can POST to and get structured JSON back. Covers data extraction, classification, summarization, and more.

Best for:
- Seeing how complex agents fit in ~20 lines of YAML
- Learning the `POST /api/v1/run` pattern
- Adapting a pattern to your own data

## See also

- [Quickstart](/getting-started/quickstart) - build your first workflow in 5 minutes
- [Workflow mode](/modes/workflow-mode) - deterministic DAG execution
- [Agent mode](/modes/agent-loop-mode) - interactive LLM-driven tool calling
