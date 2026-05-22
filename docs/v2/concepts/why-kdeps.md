# Why kdeps?

## The Problem

Shipping AI into production means more than calling an API. You need deterministic pipelines, typed inputs and outputs, dependency ordering, retries, validation, and the ability to deploy anywhere - not a chat session that ends when the browser tab closes.

kdeps is an **AI appliance builder**. You define what the agent does in YAML, and it runs as a self-contained unit: an HTTP API, a scheduled job, a bot, a file processor - without a human in the loop.

## Two Modes

| Mode | Command | Use case |
|---|---|---|
| **Workflow** | `kdeps run` | Deterministic DAG pipeline. Inputs arrive, resources execute in dependency order, output is returned. Ships to production. |
| **Agent** | `kdeps serve` | Interactive LLM loop. Every resource is auto-registered as a tool. The LLM calls them to complete tasks. |

The same `workflow.yaml` runs in both. Workflow mode is for production - autonomous, event-driven, predictable. Agent mode is for interactive use - a chat interface backed by your custom toolset instead of a generic model.

## Defined Control Flow

Chat interfaces are deliberately open-ended. kdeps workflow mode is the opposite: inputs are declared, outputs are typed, dependencies are explicit, and validations are enforced before any LLM is called. If something is wrong, it fails fast with a clear error rather than hallucinating a response.

This is what makes workflow output reproducible, auditable, and safe to run unattended.

## Agencies

Single-agent workflows are often insufficient for complex logic. kdeps lets you orchestrate **Agencies** - collections of specialized agents that coordinate and delegate tasks via the `agent:` resource type.

- Each agent can be locked to specific models and tools optimized for its task
- Agents communicate through a fully defined control flow
- Every step is visible, version-controlled, and testable

## Who it is for

- **Marketing and growth teams** automating content generation, SEO pipelines, social media publishing, lead scoring, and campaign reporting
- **Operations teams** eliminating repetitive manual work - data entry, report generation, invoice processing, email triage, document summarization
- **Developers** shipping AI features into products (APIs, bots, internal tools) without glue code
- **Any team** that has a workflow currently done by a human clicking through tabs and copy-pasting between tools
- Engineers who need agent logic in version control - reviewable, testable, and reproducible across environments

## Who it is not for

- One-off Q&A or research - use a chat interface directly
- No-code AI builders - kdeps is YAML infrastructure, not a drag-and-drop tool
