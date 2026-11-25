# kdeps deploy

kdeps deploy ships the workflow that you tested locally, with no change. It packages the workflow and builds an image with no Dockerfile. The same `workflow.yaml` runs as a Docker container, a Kubernetes deployment, a bootable ISO, or one self-contained binary.

**Not this?** To deploy a shared model server instead of an agent, see [kdeps LLM server](/llm-server/). To publish an agent for others to install, see [kdeps registry](/registry/).

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

## Targets

| Target | Page |
|---|---|
| Docker image | [Docker deployment](/deploy/docker) |
| Kubernetes manifests | [Kubernetes deployment](/deploy/kubernetes) |
| One binary, no Docker | [Standalone binaries](/deploy/binaries) |
| Web frontend next to the API | [Web server mode](/deploy/webserver) |
| HTTPS for a custom domain | [TLS and HTTPS](/deploy/tls-https) |

## Read next

- [kdeps deploy reference](/reference/deploy/) - each step, a full CI/CD pipeline, binaries, the LLM server appliance
- [Packaging commands](/reference/deploy/cli) - every `kdeps bundle` and `kdeps export` flag
