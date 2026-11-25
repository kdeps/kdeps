# kdeps deploy

kdeps deploy ships the workflow that you tested locally, with no change. The same `workflow.yaml` runs as a Docker container, a Kubernetes deployment, a bootable ISO, or one self-contained binary.

```d2
direction: right

A: "1. Validate\nkdeps validate"
B: "2. Package\nkdeps bundle package"
C: "3. Build\nkdeps bundle build"
D: "4. Run\nDocker, Kubernetes,\nISO or binary"

A -> B -> C -> D
```

```bash
kdeps validate workflow.yaml
kdeps bundle package .                     # -> my-agent-1.0.0.kdeps
kdeps bundle build my-agent-1.0.0.kdeps    # -> Docker image kdeps-my-agent:1.0.0
```

## Read next

- [kdeps deploy reference](/reference/deploy/) - every key, option and detail
