---
title: What is kdeps?
description: kdeps in plain English - what it is, the problem it solves, and the smallest mental model before you run anything.
---

# What is kdeps?

kdeps is a **git-native AI appliance builder**. You describe an agent in YAML -
which model to call, what to validate, what shape the answer takes - and those
files are the whole spec. Commit them, and kdeps packages the workflow, its
tools, and the model into one self-contained unit you can run as a terminal
REPL, an HTTP API, a Docker image, a Kubernetes deployment, a bootable ISO, or a
single binary. The same files, no rewrite, no framework to import.

Change the YAML and the appliance behaves differently - your git history is the
changelog of the agent's behavior. Because it runs open-source models by default,
the unit has no per-token cost and no dependency on an external AI service; it
works the same on your laptop and inside an air-gapped network.

## The problem it solves

Calling an LLM is easy. Shipping that call as something you can review, version,
and run inside your own boundary is not. You end up hand-writing the same glue
every time: input validation, retries, ordering between steps, a fixed response
schema, a container, a way to run it offline for tests. kdeps is that glue,
declared in YAML that lives in your repo.

The result is one deployable unit - REPL, API, Docker image, ISO, or binary -
with the model packaged in. It runs open-source models by default (llamafile,
Ollama, or any HuggingFace GGUF), so there is no per-token bill and no
third-party AI subscription. Cloud providers (OpenAI, Anthropic, Groq) still
work when you want them; the backend is one line of machine-local config, not
part of the repo.

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
- A **mode** is how you run the workflow. [Workflow mode](/workflow/)
  (`kdeps run`) executes the steps in a fixed order and returns a structured
  response - this is what you deploy. [Agent mode](/agent/)
  (`kdeps [path]`) starts a chat REPL where an LLM decides when to call the
  workflow.

Every term kdeps uses is defined in the [Glossary](/reference/glossary).

## The smallest working example

No YAML at all - just run the binary:

```bash
kdeps            # opens an AI chat REPL against a local model, no API key
```

A minimal workflow is a folder with a manifest and two resources - one that
calls the model, one that shapes the response:

```yaml
# my-agent/workflow.yaml - the manifest
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  targetActionId: response   # which resource produces the final result
```

<div v-pre>

```yaml
# my-agent/resources/llm.yaml - call the model
actionId: llm
chat:
  model: llama3.2:1b         # a local model, downloaded on first run
  prompt: "{{ get('q') }}"   # 'q' comes from the HTTP request body or REPL input
```

</div>

```yaml
# my-agent/resources/response.yaml - shape the reply
actionId: response
requires: [llm]              # runs after llm
apiResponse:
  success: true
  response:
    answer: get('llm').message.content
```

```bash
kdeps run ./my-agent/     # run it once / serve it
kdeps ./my-agent/         # or load it as a tool in the chat REPL
```

## Which product do you need?

kdeps is a small number of bounded things. Most people need one or two.

| You want to... | Product |
|---|---|
| A local autonomous agent - tool use, memory, a REPL | [kdeps agent](/agent/) |
| A deterministic YAML pipeline - API, bot, or file processor | [kdeps workflow](/workflow/) |
| To orchestrate several agents/workflows as one system | [kdeps agencies](/agencies/) |
| Just a self-hosted OpenAI-compatible endpoint, no workflow | [kdeps LLM server](/llm-server/) |
| To ship a tested workflow as Docker / K8s / ISO / a binary | [kdeps deploy](/deploy/) |
| To find, install, or publish shared agents by name | [kdeps registry](/registry/) *(optional)* |

The registry is the only optional piece - it shares agents by name, it is not a
step in building or running your own.

## First steps

| You want to... | Start here |
|---|---|
| Run an AI agent locally right now | [Run locally in 30 seconds](/agent/quickstart) |
| Understand why kdeps works this way | [Why kdeps?](/start/why-kdeps) |
| Build a real HTTP API from YAML | [Quickstart](/workflow/quickstart) |
| See the full picture of every concept | [Concepts overview](/start/concepts) |
