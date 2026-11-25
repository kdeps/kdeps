# kdeps workflow

A kdeps workflow is a deterministic YAML pipeline. A request arrives, resources run in dependency order, and kdeps returns the result. The same input always has the same execution path, so the workflow is auditable and testable. Serve it as an HTTP API, a bot, a file processor, or an interactive prompt.

```bash
kdeps run ./my-agent/      # run once / serve on :16395
```

A `chat` with `loop:` in this file is an agent. To run several workflows as one system, see [Agencies](/agencies/). To ship the file, see [Deploy](/deploy/).

## How it works

```d2
direction: down

A: "incoming request\nPOST /api/v1/chat" {shape: oval}
B: "resolve dep graph\nwalks backward from targetActionId"
C: "resource: validate\nfails fast if input invalid"
D: "resource: llm\nreads get('q'); calls the model"
E: "resource: resp\nreads get('llm'); builds the response"
F: HTTP response {shape: oval}

A -> B -> C
C -> D: "output stored as get('validate')"
D -> E: "output stored as get('llm')"
E -> F
```

A resource runs only after everything in its `requires:` has produced output. Resources with no shared dependency path run concurrently.

## Input sources

| Source | Set in `deploy` | Runs |
|---|---|---|
| API (default) | `apiServer:` | an HTTP server |
| Bot | `input: {sources: [bot]}` | a chat-platform bot |
| File | `input: {sources: [file]}` | once on one file or stdin, then exits |
| Interactive | `input: {sources: [interactive]}` | a terminal prompt: each line runs the workflow |

## Read next

- [Workflow quickstart](/workflow/quickstart) - a two-resource chat API, step by step
- [kdeps workflow reference](/reference/workflow/) - input source settings, agent memory (`--memory`), security of the API routes
- [Resource catalog](/workflow/resources) - every resource type
