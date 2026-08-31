# Examples

Complete, runnable workflows that demonstrate common patterns. Every example is copy-paste ready - clone, add your API keys, and run.

## Overview

Each example targets a different use case. Pick the one closest to what you're building:

| Example | Mode | What it demonstrates |
|---|---|---|
| [Document summarizer](/examples/file-processor) | Workflow | The `file` input source - read one file, return JSON, exit |
| [Batch processing](/examples/batch-processing) | Workflow | `items:` iteration - process a list in one request |
| [Document search (RAG)](/examples/rag-search) | Workflow | `embedding:` upsert and search, two routes in one workflow |
| [Web scraper](/examples/web-scraper) | Workflow | `scraper:` fetch, LLM summary, `jsonResponse` |
| [Login and sessions](/examples/session-auth) | Workflow | SQLite sessions, `set`/`get` session scope, auth gate |
| [Shell command API](/examples/shell-command-api) | Workflow | `exec:` with a timeout, method scoping, `info()` |
| [Function calling](/examples/function-calling) | Workflow | LLM `tools:`, `script:`, tool arguments in `memory` |
| [Image analysis](/examples/vision) | Workflow | Multipart upload, `files:` on a chat prompt, Ollama vision |
| [SQL-backed API](/examples/sql-api) | Workflow | `sql:` parameterized queries, CSV output, batch transactions |
| [Chat web app](/examples/chat-web-app) | Workflow | API + static frontend together, `public` routes, `scenario:` |
| [File upload](/examples/file-upload) | Workflow | Multipart uploads, `info('files')`, `get(field, 'filepath')` |
| [Conditionals and lists](/examples/control-flow) | Workflow | Ternary, `&&`/`||`/`!`, `filter`/`map`/`all`/`any` |
| [Static site](/examples/static-site) | Workflow | Web server mode, static file serving |
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

## Web scraper

An API that takes a URL, fetches the page with the built-in `scraper:`
resource, and returns an LLM summary as JSON.

Best for:
- Turning a page into structured data
- Feeding scraped text into a larger pipeline

[Build it step by step](/examples/web-scraper)

## Login and sessions

An API with a `POST /login` endpoint that starts a SQLite-backed session and a
`GET /session` endpoint that only works for a logged-in caller.

Best for:
- Adding authentication to a workflow
- Learning session storage scopes

[Build it step by step](/examples/session-auth)

## Shell command API

An endpoint that runs a shell command with `exec:` and returns its output -
the pattern for exposing a script or CLI tool as an HTTP service.

Best for:
- Wrapping an existing script
- System checks and automation endpoints

[Build it step by step](/examples/shell-command-api)

## Function calling

An API where the LLM calls your own resources mid-response - a `python:`
calculator and a mock database lookup - then answers using the results.

Best for:
- Letting a model compute or look things up instead of guessing
- Learning the `tools:` / `script:` pattern

[Build it step by step](/examples/function-calling)

## Image analysis

An API that accepts an image upload and a question, sends both to a multimodal
model, and returns a structured description.

Best for:
- Vision use cases (captioning, object listing, scene classification)
- Learning file uploads and `files:` on a chat prompt

[Build it step by step](/examples/vision)

## SQL-backed API

A two-route API over SQLite: `GET /report` runs an analytics query and returns
CSV; `POST /update` applies a batch of writes in a transaction.

Best for:
- Exposing a database read or write as an HTTP endpoint
- Learning parameterized queries and `paramsBatch:`

[Build it step by step](/examples/sql-api)

## Chat web app

One project that serves a chat API and the static web page that calls it. The
API route is `public` so the browser needs no token; the LLM has a system
prompt via `scenario:` and a friendly `onError` fallback.

Best for:
- A complete, deployable chat UI in one repo
- Learning `apiServer:` + `webServer:` together

[Build it step by step](/examples/chat-web-app)

## File upload

An endpoint that accepts one or more multipart uploads and returns their
metadata - count, names, MIME types, and the first file's path on disk.

Best for:
- Accepting user files
- The first step of any upload-then-process pipeline

[Build it step by step](/examples/file-upload)

## Conditionals and lists

One resource that exercises every expression control-flow tool: the ternary
operator, `&&` / `||` / `!`, and `filter` / `map` / `all` / `any` over a list.

Best for:
- A reference for expression syntax you can run and poke at

[Build it step by step](/examples/control-flow)

## Static site

Serve a folder of HTML, CSS, and JS over HTTP with web server mode. No
resources, no LLM.

Best for:
- A frontend in front of an agent API
- A docs site or dashboard

[Build it step by step](/examples/static-site)

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
