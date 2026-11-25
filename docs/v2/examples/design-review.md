# Design review panel

Four reviewer resources check a design document in parallel. A rejection ends the reply with a question. The next request answers it, and the session carries the findings across.

```d2
direction: right
req: "POST /api/v1/review\n{doc}" {shape: oval}
req -> scope
req -> clarity
req -> failure-modes
req -> readiness
scope -> reply
clarity -> reply
failure-modes -> reply
readiness -> reply
reply -> out: "passed, findings\nquestion + options" {shape: oval}
```

## Read next

- [Full walkthrough](/reference/examples/design-review) - every step, file and command
