# Deploy

Same workflow files you run with **`kdeps run`**. Packaging CLI lives under **`kdeps bundle`**.

## Package

```bash
kdeps validate .
kdeps bundle package .
# -> name-version.kdeps
kdeps run name-1.0.0.kdeps
```

## Docker

```bash
kdeps bundle build .
kdeps bundle build . --tag myregistry/myagent:latest
kdeps bundle build . --gpu cuda --tag myregistry/myagent:gpu
kdeps bundle build . --show-dockerfile
```

Pass runtime secrets via env (`KDEPS_API_AUTH_TOKEN`, provider keys) — not baked into YAML.

## Kubernetes / ISO

```bash
kdeps export k8s . -o deploy.yaml
kdeps export iso …
# also: kdeps bundle export …
```

Use `kdeps export --help` / `kdeps bundle export --help` for flags on your build.

## Preflight

```bash
kdeps validate .
kdeps doctor
```

## TLS

[TLS / HTTPS](/tls) · [Security](/security) · [CLI](/cli).
