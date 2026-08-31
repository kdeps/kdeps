# Use an MCP server as a tool

*Applies to workflow mode.*

## Overview

In this tutorial you give an LLM tools that are backed by an external
[Model Context Protocol](https://modelcontextprotocol.io) (MCP) server instead
of your own resources. kdeps starts the server as a subprocess, performs the
handshake, calls the tool, and shuts it down.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart) and read
[Tools (function calling)](/concepts/tools). It assumes you know:

- Basic YAML
- What an MCP server is at a high level

By the end you will be able to:

- Define a tool with an `mcp:` block instead of `script:`
- Point it at an MCP server started with `npx` or `uvx`
- Restrict the server's filesystem access with its arguments

## Background

A workflow-mode tool normally runs one of your resources (`script:`). An `mcp:`
tool runs a tool exposed by an MCP server. `mcp:` and `script:` are mutually
exclusive. A fresh subprocess is started for each tool invocation and only
`stdio` transport is supported.

## Before you start

- kdeps installed (`kdeps --version`).
- Node.js with `npx` on `PATH` (for the filesystem MCP server).
- A working directory for the project.

## Step 1: create the project

```bash
mkdir mcp-chat
cd mcp-chat
mkdir resources
```

## Step 2: define the route

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: mcp-chat
  version: "1.0.0"
  targetActionId: chat

settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /chat
        methods: [POST]
```

## Step 3: give the LLM MCP tools

Create `resources/chat.yaml`:

<div v-pre>

```yaml
# resources/chat.yaml
actionId: chat
name: Chat with MCP filesystem tools
validations:
  methods: [POST]
  routes: [/chat]
chat:
  model: llama3.2:1b
  role: user
  prompt: "{{ input('message') }}"
  system: |
    You are an assistant with filesystem tools via MCP. Use read_file to read
    files and list_directory to explore. Only access paths the user names.
  tools:
    - name: read_file
      description: "Read the contents of a file at the given path."
      mcp:
        server: npx
        args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
        transport: stdio        # the only supported transport
      parameters:
        path:
          type: string
          description: "Absolute path to the file to read."
          required: true
    - name: list_directory
      description: "List files and subdirectories in a directory."
      mcp:
        server: npx
        args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
      parameters:
        path:
          type: string
          description: "Absolute path to the directory to list."
          required: true
apiResponse:
  success: true
  response:
    reply: "{{ output('chat') }}"
```

</div>

The last argument to `server-filesystem` (`/tmp`) is the only directory the
server will touch - the model cannot read outside it.

## Step 4: validate and run

```bash
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
echo "hello from mcp" > /tmp/note.txt
kdeps run .
```

```bash
curl -X POST http://localhost:16395/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "What is in /tmp/note.txt?"}'
```

The model calls `read_file` with `path: "/tmp/note.txt"`, kdeps runs the MCP
server, returns the contents, and the model answers.

## Summary

You gave an LLM tools that:

- Are backed by an external MCP server, not your own resources
- Use an `mcp:` block with `server`, `args`, and `transport`
- Are sandboxed to one directory by the server's arguments

## Next steps

- [Tools (function calling)](/concepts/tools) - `script:` tools, parameter types
- [Tools reference](/reference/tools-reference) - MCP details, debugging
- [Function calling tutorial](/examples/function-calling) - tools backed by your own resources
- [Agent loop built-in tools](/modes/agent-loop-tools) - MCP in agent mode
