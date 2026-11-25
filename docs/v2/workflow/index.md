# kdeps workflow

A kdeps workflow is a deterministic YAML pipeline. A request arrives, resources run in dependency order, and kdeps returns the result. Serve it as an HTTP API, a bot, a file processor, or an interactive prompt.

```bash
kdeps run ./my-agent/      # run once / serve on :16395
```

## Read next

- [kdeps workflow reference](/reference/workflow/) - every key, option and detail
- [Workflow quickstart](/workflow/quickstart) - a two-resource chat API, step by step
- [Resource catalog](/workflow/resources) - every resource type
