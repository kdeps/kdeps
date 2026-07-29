# CLI

Primary surface: **coding agent** (`kdeps` / `kdeps [path]`).  
Secondary: **workflow** (`kdeps run`, validate, bundle).  
Also: registry, LLM appliances, exports.

Prefer `kdeps <cmd> --help` on your binary (docs track **v2.1.11**).

## Coding agent

```bash
kdeps
kdeps .
kdeps ./my-agent/ --model deepseek-v4-flash --system "You ship carefully."
kdeps --resume <session-id>
kdeps --skill ~/.kdeps/skills/
```

| Flag | Purpose |
|------|---------|
| `[path]` | Workflows / agencies under path → tools |
| `--model` | Model id |
| `--backend` | Backend id |
| `--base-url` | OpenAI-compat base URL |
| `--system` | System prompt |
| `--resume` | Session id |
| `--skill` | Skill path (repeatable) |
| `--debug` / `--verbose` / `--instrument` | Diagnostics |

Inside the REPL: `/help`, `/model`, `/goal`, `/session`, `!cmd`. See [Coding agent](/agent).

## Develop (workflow)

```text
kdeps new my-agent
  -> edit workflow.yaml + resources/
  -> kdeps validate .
  -> export KDEPS_API_AUTH_TOKEN=...
  -> kdeps run workflow.yaml [--dev]
  -> (optional) kdeps bundle package|build|export
```

| Command | Purpose |
|---------|---------|
| `kdeps new <name>` | Scaffold (`-t` template, default `api-service`) |
| `kdeps new <name> --force` | Overwrite dir |
| `kdeps validate <path>` | Schema, deps, expressions |
| `kdeps run <workflow.yaml\|.kdeps>` | Run workflow |
| `kdeps edit` | Edit `~/.kdeps/config.yaml` |
| `kdeps env` | Print connection env exports |
| `kdeps doctor` | Environment checks |
| `kdeps chat` | Chat helper (`--help`) |

### `kdeps run`

| Flag | Purpose |
|------|---------|
| `--port <n>` | Listen port (default `16395`) |
| `--dev` | Hot reload |
| `--file <path>` | File input (overrides stdin / `KDEPS_FILE_PATH`) |
| `--events` | NDJSON events on stderr |
| `--interactive` | Workflow + agent REPL |
| `--memory` | Workflow memory expression helpers |

```bash
kdeps run workflow.yaml --dev --port 16395
kdeps run workflow.yaml --interactive
kdeps run myapp-1.0.0.kdeps
```

### `kdeps validate`

Accepts workflow file, agent dir, component dir, or agency dir.

```bash
kdeps validate workflow.yaml
kdeps validate examples/chatbot
```

## Package

Under **`kdeps bundle`**:

| Command | Purpose |
|---------|---------|
| `kdeps bundle package <dir>` | `.kdeps` / `.kagency` archive |
| `kdeps bundle build <path>` | Docker image |
| `kdeps bundle prepackage …` | Standalone binary packaging |
| `kdeps bundle export …` | Bundle exports |

Top-level deploy export:

| Command | Purpose |
|---------|---------|
| `kdeps export iso\|k8s …` | ISO / Kubernetes |

```bash
kdeps bundle package .
kdeps bundle build . --tag myregistry/agent:latest --gpu cuda
kdeps export k8s . -o deploy.yaml
```

## Registry

| Command | Purpose |
|---------|---------|
| `kdeps registry search <q>` | Find packages |
| `kdeps registry install <pkg>` | Install |
| `kdeps registry list` | Installed |
| `kdeps registry info <pkg>` | Metadata |

## LLM appliances

Standalone inference only — no workflow path. [LLM server](/llm-server).

```bash
kdeps llm wizard|list|models|show|build|run|export|client-config
kdeps llamafile list|update
```

## Global flags

`--debug`, `--verbose`, `--instrument`, `-v` / `--version`.

Next: [Coding agent](/agent) · [Workflow mode](/workflow).
