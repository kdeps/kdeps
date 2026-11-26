# Use an MCP server as a tool

Give an LLM tools from an external
[Model Context Protocol](https://modelcontextprotocol.io) (MCP) server, not
from your own resources. kdeps starts the server as a subprocess, performs the
handshake, calls the tool, and shuts it down.

A tool can come from an MCP server:

```yaml
# resources/chat.yaml (part)
chat:
  tools:
    list:
      - name: read_file
        description: "Read the contents of a file at the given path."
        mcp:
          server: npx
          args: ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
          transport: stdio
```

## Read next

- [Full walkthrough](/reference/examples/mcp-tools) - every step, file and command
- [Function calling](/workflow/tools) - `script:` tools, parameter types
- [Function calling tutorial](/examples/function-calling) - tools backed by your own resources
