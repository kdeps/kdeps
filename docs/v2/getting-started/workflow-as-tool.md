# Load a workflow as a tool

Turn a `kind: Workflow` file into an LLM-callable tool in agent mode. You type a prompt; the model decides when to call the workflow; kdeps runs the full DAG and returns the result.

*Applies to agent mode.*

## Overview

This tutorial is for developers who have:

- Run the [agent REPL](/getting-started/local-agent)
- Built the [quickstart](/getting-started/quickstart) HTTP API (or any `workflow.yaml`)

By the end you will load that workflow as a tool named after `metadata.name`, call it from a prompt, and see the DAG run as one tool invocation.

## Before you start

- kdeps installed (`kdeps --version`)
- A workflow directory with `workflow.yaml` and a `metadata.name` (the quickstart project `my-agent` is enough)

No extra install. The same local llamafile from the REPL is used.

## How it works

```text
you type a prompt
        |
        v
LLM calls tool "my-agent"
        |
        v
kdeps runs the full workflow DAG
```

The LLM never sees the resource YAML. It sees one tool, named `metadata.name`, and the tool's output (`apiResponse.response`).

## Step 1: confirm the tool name

Open `workflow.yaml` and read `metadata.name`. That string is the tool name. In the quickstart it is `my-agent`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent          # tool name in agent mode
  version: "1.0.0"
  targetActionId: response
```

There is no `kind: Agent`. Agent mode is how you *run* a workflow, not a separate YAML kind.

## Step 2: start the REPL with the workflow

From the project directory:

```bash
kdeps .
```

Or pass the path:

```bash
kdeps ./my-agent/
```

The REPL starts. The model can call one workflow tool (`my-agent`) plus the built-in tools (search, shell, files, memory). It does not register individual resources as tools.

## Step 3: ask something the workflow can answer

Type a prompt that needs the workflow. For the quickstart chat API:

```text
> Summarize entropy in one sentence using my-agent
```

The model calls `my-agent`. kdeps runs every resource in `requires:` order and returns `apiResponse.response` to the model. The model then answers you.

A folder of workflows registers one tool per `workflow.yaml` (and per agency):

```bash
kdeps ./agents/
```

## Summary

- `kdeps [path]` is agent mode. `kdeps run workflow.yaml` is workflow mode. Same file.
- The tool name is `metadata.name`, not the filename.
- The unit of work is the whole workflow, not a single resource.

## Next steps

- [Agent mode](/modes/agent-loop-mode) - flags, folder discovery, how tools run
- [Built-in tools](/modes/agent-loop-tools) - what the REPL can do with no YAML
- [Skills and prompt templates](/modes/agent-loop-skills) - markdown that shapes the REPL
- [Quickstart](/getting-started/quickstart) - the HTTP API for the same file
