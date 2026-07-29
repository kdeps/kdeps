# CLI

High-signal commands. Always prefer `kdeps <cmd> --help` for flags on your build.

## Everyday

| Command | Purpose |
|---------|---------|
| `kdeps` | Agent REPL |
| `kdeps [path]` | Agent REPL with workflows as tools |
| `kdeps run <workflow.yaml>` | Workflow mode |
| `kdeps run <workflow.yaml> --dev` | Reload on change |
| `kdeps new <name>` | Scaffold a project |
| `kdeps validate <workflow.yaml>` | Schema / graph checks |
| `kdeps doctor` | Environment checks |

## Bundle and deploy

| Command | Purpose |
|---------|---------|
| `kdeps bundle package …` | Build `.kdeps` archive |
| `kdeps bundle build …` | Docker image from package |
| `kdeps bundle export …` | k8s / iso / binary exports |

## Registry

| Command | Purpose |
|---------|---------|
| `kdeps registry install <name>` | Install a component |
| `kdeps registry search …` | Find components |
| `kdeps registry list` | Installed / available |

## LLM appliances

No workflow path. See [LLM server](/llm-server).

| Command | Purpose |
|---------|---------|
| `kdeps llm` / `wizard` | Interactive builder |
| `kdeps llm list` | Recipes |
| `kdeps llm models` | Harvested llamafile / GGUF models |
| `kdeps llm show <id>` | Recipe detail |
| `kdeps llm client-config --url …` | Client snippet |
| `kdeps llm build / run / export` | Image, container, k8s, ISO |
| `kdeps llamafile update` | Refresh model harvest |

## Local models

```bash
kdeps llamafile list
kdeps llamafile update
```

In the agent REPL, `/model` and `/model ps` manage active local servers.
