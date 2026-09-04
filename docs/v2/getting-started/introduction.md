---
title: What is kdeps?
description: kdeps in plain English - what it is, the problem it solves, and the smallest mental model before you run anything.
---

# What is kdeps?

kdeps is a single binary that turns a folder of YAML files into an AI agent you
can run two ways: as an interactive chat in your terminal, or as an HTTP API you
deploy to a server. The same files work both ways with no rewrite.

You describe *what the agent does* - which model to call, what to validate, what
shape the answer takes - in YAML. kdeps runs it. There is no application code to
write and no framework to import.

## The problem it solves

Calling an LLM API is easy. Shipping that call into production is not. You end up
hand-writing the same glue every time: input validation, retries, ordering
between steps, a fixed response schema, a way to deploy it, a way to run it
offline for testing. kdeps is that glue, defined declaratively and reused across
every agent you build.

## The smallest mental model

```text
a folder with workflow.yaml
        |
        +--  kdeps run ./my-agent/     ->  one-shot pipeline / HTTP API
        |
        +--  kdeps ./my-agent/         ->  interactive REPL, agent calls the workflow as a tool
```

- A **resource** is one step - an LLM call, a shell command, a SQL query, an HTTP
  request. It lives in its own YAML file.
- A **workflow** is a folder of resources plus a `workflow.yaml` manifest. Each
  resource declares what it `requires:`, and kdeps runs them in that order.
- A **mode** is how you run the workflow. [Workflow mode](/modes/workflow-mode)
  (`kdeps run`) executes the steps in a fixed order and returns a structured
  response - this is what you deploy. [Agent mode](/modes/agent-loop-mode)
  (`kdeps [path]`) starts a chat REPL where an LLM decides when to call the
  workflow.

Every term kdeps uses is defined in the [Glossary](/reference/glossary).

## The smallest working example

No YAML at all - just run the binary:

```bash
kdeps            # opens an AI chat REPL against a local model, no API key
```

A minimal one-step workflow is a folder with two files:

```yaml
# my-agent/workflow.yaml - the manifest
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  targetActionId: answer   # which resource produces the final result
```

<div v-pre>

```yaml
# my-agent/resources/answer.yaml - the one step
actionId: answer           # this resource's id, referenced by targetActionId
chat:
  model: llama3.2:1b       # a local model, downloaded on first run
  prompt: "{{ get('q') }}"   # 'q' comes from the HTTP request body or REPL input
```

</div>

```bash
kdeps run ./my-agent/     # run it once / serve it
kdeps ./my-agent/         # or load it as a tool in the chat REPL
```

## Where to go next

| You want to... | Start here |
|---|---|
| Run an AI agent locally right now | [Run locally in 30 seconds](/getting-started/local-agent) |
| Understand why kdeps works this way | [Why kdeps?](/concepts/why-kdeps) |
| Build a real HTTP API from YAML | [Quickstart](/getting-started/quickstart) |
| See the full picture of every concept | [Concepts overview](/concepts/overview) |
