# CLI

Developer surface for **workflow mode**. Prefer `kdeps <cmd> --help` on your binary.

## Daily loop

```text
kdeps new my-agent
  -> edit workflow.yaml + resources/
  -> kdeps validate .
  -> export KDEPS_API_AUTH_TOKEN=...
  -> kdeps run workflow.yaml [--dev]
  -> curl / api client
  -> kdeps bundle package|build|export   # when shipping
```

## Develop

| Command | Purpose |
|---------|---------|
| `kdeps new <name>` | Scaffold agent (`-t` / `--template`, default `api-service`) |
| `kdeps new <name> --force` | Overwrite existing directory |
| `kdeps validate <path>` | YAML, schema, deps, expressions (file or dir) |
| `kdeps run <workflow.yaml\|package.kdeps>` | Run workflow locally |
| `kdeps edit` | Open `~/.kdeps/config.yaml` |
| `kdeps env` | Print env exports for configured connections |
| `kdeps doctor` | Health checks (config, models, Python, …) |
| `kdeps chat` | Chat helper (see `--help`) |

### `kdeps run` flags

| Flag | Purpose |
|------|---------|
| `--port <n>` | Listen port (default `16395`) |
| `--dev` | Hot reload |
| `--file <path>` | File input source path (overrides stdin / `KDEPS_FILE_PATH`) |
| `--events` | NDJSON lifecycle events on stderr |
| `--interactive` | Run workflow **and** open LLM REPL alongside it |
| `--memory` | Enable workflow memory expression helpers |
| `--debug` / `--verbose` | Logging (persistent flags) |

```bash
kdeps run workflow.yaml
kdeps run workflow.yaml --dev --port 16395
kdeps run workflow.yaml --file ./doc.txt
kdeps run workflow.yaml --interactive
kdeps run myapp-1.0.0.kdeps
```

API routes need a token (not in git):

```bash
export KDEPS_API_AUTH_TOKEN=dev-token
# or: api_auth_token in ~/.kdeps/config.yaml
```

### `kdeps validate` accepts

- `workflow.yaml` path
- Agent dir (has `workflow.yaml`)
- Component dir (`component.yaml`)
- Agency dir (`agency.yaml`)

```bash
kdeps validate workflow.yaml
kdeps validate examples/chatbot
```

## Package (`kdeps bundle …`)

Top-level packaging lives under **`bundle`**.

| Command | Purpose |
|---------|---------|
| `kdeps bundle package <dir>` | `.kdeps` / `.kagency` archive |
| `kdeps bundle build <path>` | Docker image (dir, yaml, or package) |
| `kdeps bundle prepackage …` | Standalone binary packaging |
| `kdeps bundle export …` | Export formats under bundle |

Also top-level:

| Command | Purpose |
|---------|---------|
| `kdeps export iso\|k8s …` | Deploy-group export (ISO, Kubernetes) |

```bash
kdeps bundle package my-agent/
kdeps bundle package my-agent/ --output dist/
kdeps bundle build my-agent/ --tag myregistry/myagent:latest
kdeps bundle build my-agent/ --gpu cuda --show-dockerfile
kdeps export k8s my-agent/ -o deploy.yaml
```

## Registry

| Command | Purpose |
|---------|---------|
| `kdeps registry search <q>` | Find packages |
| `kdeps registry install <pkg>` | Install component / agent |
| `kdeps registry list` | Installed |
| `kdeps registry info <pkg>` | Metadata |

## Agent REPL (not workflow)

| Command | Purpose |
|---------|---------|
| `kdeps` | Model REPL |
| `kdeps [path]` | REPL; workflows under path become tools |
| `kdeps --model … --system …` | Model / system prompt |
| `kdeps --resume <id>` | Resume session |
| `kdeps --skill <path>` | Load skills |

See [Agent mode](/agent).

## LLM appliances (not workflow)

No workflow path argument. See [LLM server](/llm-server).

```bash
kdeps llm list|models|show|build|run|export|client-config|wizard
kdeps llamafile list|update
```

## Global flags

| Flag | Purpose |
|------|---------|
| `--debug` | Debug logging |
| `--verbose` | Verbose output |
| `--instrument` | Call-chain tracing |
| `-v` / `--version` | Version |

Agent-only root flags: `--backend`, `--base-url`, `--model`, `--resume`, `--skill`, `--system`.

Next: [Workflow mode](/workflow) · [Quickstart](/quickstart).
