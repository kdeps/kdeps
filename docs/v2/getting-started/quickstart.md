# Quickstart

Build a two-resource LLM API in [workflow mode](/modes/workflow-mode), then load the same file as a tool in [agent mode](/modes/agent-loop-mode).

*Applies to both workflow mode and agent mode. New to kdeps? Read [What is kdeps?](/getting-started/introduction) first.*

## The mental model

A **resource** is one step in its own YAML file (here: an LLM call, then a
response). A **workflow** is a folder of resources plus a `workflow.yaml`
manifest. Each resource names what it `requires:`, and kdeps runs the steps in
that dependency order - a DAG. One step's output is read by the next with
`get('<actionId>')`. That fixed order is what "workflow mode" means, and it is
what you deploy.

## Overview

This quickstart guides you through:

- Creating a project
- Writing a two-resource workflow (an LLM call and a response)
- Running it as an HTTP API in workflow mode
- Calling the same workflow as a tool in agent mode

It is for developers who have used a terminal and an HTTP API before. For the
REPL with no YAML, see [Run locally](/getting-started/local-agent). For install
options (Windows, source, Docker), see [Installation](/getting-started/installation).

## Before you start

- kdeps installed (`kdeps --version`)
- No LLM server: models run as local
  [llamafiles](https://github.com/Mozilla-Ocho/llamafile) (the default `file`
  backend). The default model (`llama3.2:1b`, ~1.1 GB) is confirmed, then
  downloaded to `~/.kdeps/models/` automatically on first run.

## Create a project

```bash
kdeps new my-agent
cd my-agent
```

Or create the structure manually:

```bash
mkdir -p my-agent/resources && cd my-agent
```

## Define your workflow

`workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: response

settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

## Add an LLM resource

`resources/llm.yaml`:

<div v-pre>

```yaml
# resources/llm.yaml
actionId: llm
name: LLM Chat
validations:
  methods: [POST]
  routes: [/api/v1/chat]
  check:
    - get('q') != ''
  error:
    code: 400
    message: "'q' is required"
chat:
  model: llama3.2:1b
  role: user
  prompt: "{{ get('q') }}"
  timeout: 60s
```

</div>

## Add a response resource

`resources/response.yaml`:

```yaml
# resources/response.yaml
actionId: response
name: API Response
requires: [llm]
apiResponse:
  success: true
  response:
    # chat output is the raw response object; the reply text is at .message.content
    answer: get('llm').message.content
```

## Run it

When `apiServer` is configured, kdeps requires an API auth token before it starts. Set one for local development (never in `workflow.yaml`):

```bash
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run workflow.yaml
```

You can also set `api_auth_token` in `~/.kdeps/config.yaml`. See [Security reference](/reference/security).

Test the API:

```bash
curl -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q": "What is entropy?"}'
```

Expected response:

```json
{
  "success": true,
  "data": {
    "answer": "Entropy is a measure of disorder..."
  }
}
```

## How it works

```d2
direction: down

A: "POST /api/v1/chat\n{\"q\": \"What is entropy?\"}" {shape: oval}
B: "resource: llm\nvalidates get('q') != ''; calls llama3.2:1b"
C: "resource: response\nrequires: [llm]; reads get('llm').message.content"
D: "{\"success\": true, \"response\": {\"answer\": \"...\"}}" {shape: oval}

A -> B
B -> C: "output stored as get('llm')"
C -> D
```

`requires: [llm]` means `response` will not run until `llm` has finished. This two-resource DAG is the simplest workflow mode pipeline.

## Try agent mode

Same file, different command. `kdeps .` registers this workflow as one tool named `my-agent`. Step-by-step: [Load a workflow as a tool](/getting-started/workflow-as-tool).

```bash
kdeps .            # current directory; tool name is metadata.name
kdeps ./agents/    # one tool per workflow in the folder
```

## Next steps

- [Load a workflow as a tool](/getting-started/workflow-as-tool) - agent mode with this file
- [Workflow mode](/modes/workflow-mode) - how the DAG pipeline runs
- [Agent mode](/modes/agent-loop-mode) - the interactive LLM loop
- [workflow.yaml reference](/configuration/workflow) - every field
- [Resources overview](/resources/overview) - all resource types
- [CLI reference](/reference/cli/) - all commands and flags
