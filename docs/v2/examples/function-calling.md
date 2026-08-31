# Give an LLM tools to call

*Applies to workflow mode.*

## Overview

In this tutorial you build an API where the LLM can call your own resources
mid-response. When the model needs a calculation or a database lookup, it calls
a tool, kdeps runs the target resource, feeds the result back, and the model
continues.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- Basic Python

By the end you will be able to:

- Define a tool on a `chat:` resource with `tools:`
- Point a tool at a resource with `script:`
- Read tool arguments in the target resource with `get(name, 'memory')`
- Understand why tool resources are "unreachable" in `kdeps validate`

## Background

A tool is a function the LLM can call. In workflow mode you declare tools in
`chat.tools`: each has a name, a description the model uses to decide when to
call it, a `script:` naming the resource that runs it, and a parameter schema.
The tool resource is not in the dependency graph - it runs only when the model
calls it.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.

## Step 1: create the project

```bash
mkdir tool-chat
cd tool-chat
mkdir resources
```

## Step 2: define the route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: tool-chat
  version: "1.0.0"
  targetActionId: toolResponse

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/tools
        methods: [POST]
  agentSettings:
    pythonVersion: "3.12"
```

## Step 3: write the calculator tool

Create `resources/calc-tool.yaml`:

<div v-pre>

```yaml
# resources/calc-tool.yaml
actionId: calcTool
name: Calculator tool
python:
  script: |
    import json, math
    expression = "{{ get('expression', 'memory') }}"   # the tool argument
    allowed = {
        "__builtins__": {},
        "sqrt": math.sqrt, "sin": math.sin, "cos": math.cos, "log": math.log,
        "pi": math.pi, "e": math.e, "abs": abs, "round": round, "pow": pow,
    }
    try:
        print(json.dumps({"result": eval(expression, allowed, {})}))
    except Exception as exc:
        print(json.dumps({"error": str(exc)}))
```

</div>

The LLM's tool arguments land in `memory` scope, so `get('expression',
'memory')` reads the `expression` argument the model supplied.

## Step 4: write a mock database tool

Create `resources/db-tool.yaml`:

<div v-pre>

```yaml
# resources/db-tool.yaml
actionId: dbTool
name: Database search tool
apiResponse:
  success: true
  response:
    results:
      - id: 1
        name: "Widget"
        category: "{{ get('category', 'memory') }}"
        query: "{{ get('query', 'memory') }}"
```

</div>

## Step 5: give the tools to the LLM

Create `resources/chat.yaml`:

<div v-pre>

```yaml
# resources/chat.yaml
actionId: llmWithTools
name: LLM with tools
chat:
  model: llama3.2:1b
  role: user
  prompt: "{{ get('q') }}"
  tools:
    - name: calculate
      description: "Evaluate a math expression. Supports +, -, *, /, **, sqrt, sin, cos, log, pi, e."
      script: calcTool                 # the resource that runs this tool
      parameters:
        expression:
          type: string
          description: "e.g. '2 + 2', 'sqrt(16)', 'sin(pi/2)'"
          required: true
    - name: search_database
      description: "Search the product database."
      script: dbTool
      parameters:
        query:
          type: string
          description: "Search query"
          required: true
        category:
          type: string
          description: "Optional category filter"
          required: false
  jsonResponse: true
```

</div>

## Step 6: return the answer

Create `resources/response.yaml`:

<div v-pre>

```yaml
# resources/response.yaml
actionId: toolResponse
name: Tool response
requires: [llmWithTools]
validations:
  methods: [POST]
  routes: [/api/v1/tools]
apiResponse:
  success: true
  response:
    query: "{{ get('q') }}"
    answer: "{{ get('llmWithTools').message.content }}"
```

</div>

## Step 7: validate and run

```bash
kdeps validate .
```

Validation prints a warning: `calcTool: resource is unreachable from
targetActionId`. That is expected - tool resources are only reached when the
LLM calls them, not through `requires:`.

```bash
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run .
```

```bash
curl -X POST http://localhost:16395/api/v1/tools \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q": "What is the square root of 144, and search for blue widgets?"}'
```

The model calls `calculate` with `expression: "sqrt(144)"`, calls
`search_database` with `query: "widgets", category: "blue"`, then answers using
both results.

## Summary

You built an API where the LLM:

- Chooses between two tools based on their descriptions
- Calls `calcTool` (a `python:` resource) and `dbTool` (an `apiResponse:` resource)
- Reads its own arguments in each tool with `get(name, 'memory')`

## Next steps

- [Tools (function calling)](/concepts/tools) - MCP tools, multiple tools, parameter types
- [Tools reference](/reference/tools-reference) - tool chaining, debugging
- [Python resource](/resources/scripting/python) - building tool scripts
- [Agent loop mode](/modes/agent-loop-mode) - tools that are whole workflows
