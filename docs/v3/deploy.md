# Deploy

Same files as `kdeps run`. Packaging under **`kdeps bundle`**.

## Package → Docker

```bash
kdeps validate .
kdeps bundle package .
kdeps bundle build . --tag myregistry/agent:latest
kdeps bundle build . --gpu cuda --show-dockerfile
```

Runtime secrets via env, not committed YAML.

## Kubernetes / ISO

```bash
kdeps export k8s . -o deploy.yaml
kdeps export iso …
# also: kdeps bundle export …
```

## Standalone binary (prepackage)

Embed the archive into a self-contained executable — no Docker, no separate kdeps install.

```bash
kdeps bundle package .
kdeps bundle prepackage my-agent-1.0.0.kdeps --output dist/
# → dist/my-agent-1.0.0-linux-amd64, darwin-arm64, …
kdeps bundle prepackage my-agent-1.0.0.kdeps --arch linux-arm64
```

Copy binary to host and run it. Optional: `--include-models` for air-gapped llamafile embeds (large).

## LLM appliances only

Inference servers (no agent/workflow path): [LLM server](/llm-server) (`kdeps llm …`).

## Preflight

```bash
kdeps validate .
kdeps doctor
```

[Web server](/webserver) · [TLS](/tls) · [Security](/security) · [CLI](/cli).
