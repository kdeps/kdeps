# LLM Commands

`kdeps llm` provisions **standalone LLM server appliances**. These are not agent packages — there is **no workflow path argument**.

## kdeps llm list

List stock and user/project recipes.

```bash
kdeps llm list
```

## kdeps llm show

```bash
kdeps llm show ollama
```

## kdeps llm client-config

Print a ready-to-paste client snippet for `~/.kdeps/config.yaml` or shell env.

| Flag | Description |
|------|-------------|
| `--url` | OpenAI-compat base URL (required), e.g. `http://host:8000/v1` |
| `--api-key` | Optional bearer key |
| `--model` | Optional model allowlist (yaml only) |
| `--format` | `yaml` (default), `env`, or `export` |

```bash
kdeps llm client-config --url http://192.168.1.50:8000/v1
kdeps llm client-config --url http://llm:8000/v1 --format export
```

## Build and export

```bash
kdeps llm build --engine <id> --model <name> [--tag name:ver]
kdeps llm run --engine <id> --model <name> [-p 8000]
kdeps llm export k8s --image REG/llm:tag --engine <id>
kdeps llm export iso --engine <id> --model <name>
```

See [LLM Server Appliance](/deployment/llm-server) for architecture and recipes.
