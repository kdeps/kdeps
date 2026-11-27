# Why kdeps?

One `workflow.yaml` runs as an API, a listener, a file job, or a prompt. Each event starts one run. A `chat` with `loop:` is the step where the model picks tools. You pay for the hardware it runs on.

```d2
direction: right

api: "api\nHTTP request" {shape: oval}
listen: "listen\nchat bots, webhooks, websockets" {shape: oval}
file: "file\nstdin or path" {shape: oval}
prompt: "interactive\nterminal prompt" {shape: oval}
W: "workflow.yaml\nDAG, deterministic order"
L: "chat with loop:\nagent: tools, memory"

api -> W
listen -> W
file -> W
prompt -> W
W -> L: "when a step is an agent"
```

## Read next

- [Full text](/reference/start/why-kdeps) - the complete page
- [Quickstart](/workflow/quickstart) - build your first workflow API
- [Data sovereignty](/start/data-sovereignty) - your prompts stay on your servers
