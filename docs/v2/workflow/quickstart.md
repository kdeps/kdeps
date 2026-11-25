# Quickstart

Build a two-resource LLM API from one `workflow.yaml`. With `loop:` on the chat, the same file is an [agent loop](/workflow/agent-loop/).

```bash
kdeps new my-agent        # create the project
cd my-agent
export KDEPS_API_AUTH_TOKEN=dev-token   # local only, never in workflow.yaml
kdeps run workflow.yaml   # serve on :16395
```

## Read next

- [Full walkthrough](/reference/workflow/quickstart) - every step, file and command
- [Workflow](/workflow/) - how the pipeline runs
- [Agent loop](/workflow/agent-loop/) - a chat with `loop:`
