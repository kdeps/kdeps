# MCP Mode

MCP mode turns any workflow or agency into an MCP tool server over stdio. Every `actionId` in the workflow becomes a callable tool. Connect Claude Desktop, Cursor, Zed, or any MCP-compatible client directly to your kdeps resources.

```bash
kdeps mcp workflow.yaml          # expose workflow resources as tools
kdeps mcp ./my-agent/            # auto-discover workflow or agency in directory
kdeps mcp agency.yaml            # expose all agents in an agency as tools
kdeps mcp                        # built-in fformat tools only (no workflow)
```

## How it works

1. `kdeps mcp workflow.yaml` parses your workflow and registers each resource as an MCP tool.
2. The tool name is the resource's `actionId`.
3. The MCP client calls a tool by name, passing arguments as JSON.
4. kdeps runs the target resource with those arguments and returns the output as the tool result.
5. The server stays alive over stdio, handling one tool call at a time.

For agencies, resources across all agent workflows are registered as tools.

## When to use MCP mode

- You want to call your kdeps workflow resources from Claude Desktop, Cursor, or another MCP host.
- You are building a tool library that an LLM host should discover and call.
- You want to expose internal data sources (databases, APIs, scripts) to an external LLM without writing a separate server.

## Configuring an MCP client

### Claude Desktop

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "my-agent": {
      "command": "kdeps",
      "args": ["mcp", "/path/to/my-agent"]
    }
  }
}
```

### Cursor

Add to your `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "my-agent": {
      "command": "kdeps",
      "args": ["mcp", "/path/to/my-agent"]
    }
  }
}
```

## Tool naming and discovery

Each resource's `actionId` becomes a tool name. The resource's `name` field (if set) becomes the tool's human-readable label.

Example resource:

```yaml
actionId: searchProducts
name: Search Products
chat:
  model: llama3.2:1b
  prompt: "Find products matching: {{ get('query') }}"
  timeout: 30s
```

This resource appears to the MCP client as a tool named `searchProducts`.

## Input and output

Tool arguments passed by the MCP client become available inside the resource via `get('key')`. The resource output is returned as the tool result.

## Differences from other modes

| | Workflow mode (`kdeps run`) | Agent mode (`kdeps serve`) | MCP mode (`kdeps mcp`) |
|---|---|---|---|
| Execution | DAG, deterministic | LLM loop, tool-driven | Per-tool call, stateless |
| Entry point | `metadata.targetActionId` | User prompt | Tool name from MCP client |
| Resources | Run in order | Called on demand by LLM | Each resource = one tool |
| Session | Single execution | Interactive REPL | Stateless per call |
| Client | HTTP (curl, browser) | Terminal REPL | MCP host (Claude Desktop, Cursor) |

## See also

- [Workflow Mode](workflow-mode) - Deterministic DAG pipelines
- [Agent Mode](agent-mode) - Interactive LLM loop
