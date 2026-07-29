# Deploy

Package a workflow once, ship it many ways. LLM **appliances** (inference only) use a separate path — see [LLM server](/llm-server).

## Package

```bash
kdeps bundle package workflow.yaml
# -> myagent-1.0.0.kdeps
```

## Docker

```bash
kdeps bundle build myagent-1.0.0.kdeps --tag myregistry/myagent:latest
kdeps bundle build myagent-1.0.0.kdeps --gpu cuda --tag myregistry/myagent:gpu
kdeps bundle build myagent-1.0.0.kdeps --show-dockerfile
```

Run the image with the same env you use locally (`KDEPS_API_AUTH_TOKEN`, provider keys, etc.).

## Kubernetes

```bash
kdeps bundle export k8s myagent-1.0.0.kdeps -o deploy.yaml
# or from an image you already built
```

Apply with `kubectl`. Wire secrets via Kubernetes Secrets / env — not committed YAML.

## ISO / standalone

Bootable or binary packaging for appliance-style hosts:

```bash
kdeps bundle export iso …
kdeps bundle export binary …
```

Flags and outputs evolve — check `kdeps bundle --help` on your installed version.

## Preflight

```bash
kdeps validate workflow.yaml
kdeps doctor
```

## TLS on the host

Serve HTTPS with static PEMs or Let's Encrypt. See [TLS / HTTPS](/tls).

## Dev reload

```bash
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run workflow.yaml --dev
```

Next: [LLM server](/llm-server) · [Security](/security).
