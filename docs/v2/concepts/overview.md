# Concepts overview

The mental model behind kdeps: how a workflow is shaped, how data moves between steps, and which pieces apply to each mode. Start with the two modes, then reach for the rest as you need them.

## The two modes

| Concept | What it is | Mode |
|---|---|---|
| [Workflow mode](/modes/workflow-mode) | A DAG of resources that runs in dependency order and returns a response | Workflow |
| [Agent mode](/modes/agent-loop-mode) | An LLM that calls workflows, components, and built-in tools to finish a task | Agent |
| [Agencies](/concepts/agency) | Several agents bundled under one `agency.yaml`, calling each other with `agent:` | Both |
| [Components](/concepts/components) | Reusable resource bundles you install or build, invoked with `component:` | Both |

## Wiring data between steps

| Concept | What it is | Mode |
|---|---|---|
| [Expressions](/concepts/expressions) | expr-lang snippets in `{{ }}`, `before:`/`after:`, and `check:` | Both |
| [Expression helpers](/concepts/expression-helpers) | `Json()`, `Safe()`, `Debug()`, `default()` and other utilities | Both |
| [Data access](/concepts/unified-api) | `get()`/`set()` plus the `input` and `request` shorthands | Both (`input`/`request`: workflow) |
| [Tools (function calling)](/concepts/tools) | Let a `chat:` resource call other resources mid-response | Workflow |
| [Inline resources](/concepts/inline-resources) | `chat`/`sql`/`python`/... actions nested in a resource's `before:`/`after:` | Workflow |

## Control flow and input

| Concept | What it is | Mode |
|---|---|---|
| [Validation and control](/concepts/validation-and-control) | The `validations:` block - skip, preflight check, route/method limits, input schema | Workflow |
| [Error handling (onError)](/concepts/error-handling) | Retry, fallback value, or fail on a resource error | Workflow |
| [Items iteration](/concepts/items) | Run a resource once per list entry (for-each) | Both |
| [While-loop](/concepts/loop) | Repeat a resource while a condition holds; `every:` for scheduled tasks | Both |
| [Input sources](/concepts/input-sources) | Feed a workflow from an HTTP API, a chat bot, or a file | Workflow |
| [Jinja2 templates](/concepts/jinja2-templates) | YAML preprocessing and project scaffolding | Both |

## State

| Concept | What it is | Mode |
|---|---|---|
| [Session storage](/configuration/session) | Values that persist across requests from the same caller | Workflow |
| [Persistent memory](/concepts/memory) | Project-scoped facts the agent recalls across sessions | Agent (tools also work in workflow) |
| [Memory internals](/concepts/memory-internals) | Auto-extraction, the memory graph, and prompt injection | Agent |
