# Use an MCP server as a tool

Give an LLM tools from an external
[Model Context Protocol](https://modelcontextprotocol.io) (MCP) server, not
from your own resources. kdeps starts the server as a subprocess, performs the
handshake, calls the tool, and shuts it down.

## Read next

- [Full walkthrough](/reference/examples/mcp-tools) - every step, file and command
- [Function calling](/workflow/tools) - `script:` tools, parameter types
- [Function calling tutorial](/examples/function-calling) - tools backed by your own resources
