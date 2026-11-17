---
title: What is kdeps?
description: What kdeps is, the problem it solves, and the smallest mental model before you run anything.
---

# What is kdeps?

kdeps is a git-native AI appliance builder. You describe an agent in YAML files:
the model to call, the data to validate, and the shape of the answer. These
files are the full specification. When you commit them, kdeps packages the
workflow, its tools, and the model into one self-contained unit.

You can run this unit as an HTTP API, a terminal REPL, a Docker image, a
Kubernetes deployment, a bootable ISO, or a single binary. All of these use the
same `workflow.yaml`. You do not rewrite code, change the configuration, or
import a framework.

When you change the YAML, the appliance changes its behavior. Your git history
is the changelog of the agent behavior. By default, the unit runs open-source
models. For this reason, it has no per-token cost and does not depend on an external AI
service. It operates the same on your laptop and in an air-gapped network.

This is [data sovereignty](/start/data-sovereignty). You run your own LLM and
the workflow on servers that you control, in the region that you select. No
prompts or data go to a third-party AI provider.

## The problem it solves

A call to an LLM needs only a few lines of code. But to review, version, and
run that call in your own boundary needs much more code. Usually you write the
same code each time. This code includes input validation, retries, step order,
a fixed response schema, a container, and an offline mode for tests. kdeps
gives you this code as YAML in your repository.

You can also use cloud providers (OpenAI, Anthropic, Groq). The backend is one
line of configuration on the local machine. It is not part of the repository.

## The smallest mental model

```text
a folder with workflow.yaml
        |
        +--  kdeps            ->  runs it: HTTP API, bot, file, or interactive prompt
        |
        +--  kdeps --agent-loop   ->  the agent REPL on that folder's workflow
```

- A **resource** is one step - an LLM call, a shell command, a SQL query, an HTTP
  request. It lives in its own YAML file.
- A **workflow** is a folder of resources plus a `workflow.yaml` manifest. Each
  resource declares what it `requires:`, and kdeps runs them in that order. It is
  the one way to run kdeps.
- An **input source** is how requests reach the workflow: `api`, `bot`, `file`, or
  `interactive` (a terminal prompt). See [Input sources](/workflow/input-sources).
- An **agent** is a `chat` with [`loop:`](/workflow/resources/llm#agent-loop): the
  model calls tools over many rounds. `kdeps --agent-loop` opens it as a REPL.

The [Glossary](/reference/glossary) gives the definition of each kdeps term.

## The smallest working example

No YAML needed. Run the binary:

```bash
kdeps --agent-loop   # an AI chat REPL against a local model, no API key
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
  model:
    name: llama3.2:1b         # a local model, downloaded on first run
  context:
    prompt: "{{ get('q') }}"   # 'q' comes from the HTTP request body or REPL input
```

</div>

```yaml
# my-agent/resources/response.yaml - shape the reply
actionId: response
flow:
  requires: [llm]              # runs after llm
apiResponse:
  success: true
  response:
    answer: get('llm').message.content
```

```bash
kdeps run ./my-agent/     # run it once / serve it
kdeps --agent-loop ./my-agent/         # or load it as a tool in the chat REPL
```

## Where to read next

| You want to... | Read |
|---|---|
| Run the YAML as an API, a bot, or a file processor | [Workflow](/workflow/) |
| Open a `chat` with `loop:` as a REPL | [Agent REPL](/agent/) |
| Run several workflows as one system | [Agencies](/agencies/) |
| Serve an OpenAI-compatible endpoint, with no workflow | [LLM server](/llm-server/) |
| Ship the file as Docker, Kubernetes, an ISO, or a binary | [Deploy](/deploy/) |
| Find, install, or publish a shared agent by name | [Registry](/registry/) *(optional)* |

The registry is the only optional piece - it shares agents by name, it is not a
step in building or running your own.

## First steps

| You want to... | Start here |
|---|---|
| Build a real HTTP API from YAML | [Quickstart](/workflow/quickstart) |
| Run an AI agent locally right now | [Run locally in 30 seconds](/agent/quickstart) |
| Understand why kdeps works this way | [Why kdeps?](/start/why-kdeps) |
| See the full picture of every concept | [Concepts overview](/start/concepts) |
