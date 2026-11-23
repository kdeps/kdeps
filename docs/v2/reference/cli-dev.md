# Dev commands

Commands for local development: run, validate, scaffold, configure, and diagnose.

## `kdeps run`

Run a workflow locally (default execution mode).

```bash
kdeps run [workflow.yaml | directory | package.kdeps] [flags]
```

**Arguments:**
- `workflow.yaml` - Path to a workflow file
- `directory` - Path to a directory containing `workflow.yaml`
- `package.kdeps` - Path to packaged workflow file

**Flags:**

| Flag | Description | Default |
|---|---|---|
| `--dev` | Enable hot reload mode | `false` |
| `--port` | API server port number | From workflow config |
| `--debug` | Enable debug logging | `false` |
| `--interactive` | Open an interactive prompt alongside the running workflow | `false` |
| `--file` | File path to process (file input source only). Takes priority over stdin, `KDEPS_FILE_PATH`, and `input.file.path` config | |
| `--events` | Emit structured NDJSON execution events to stderr (resource lifecycle, failure classification) | `false` |
| `--memory` | Enable the agent memory store for a workflow. Resources can use `memory_save`, `memory_search`, `memory_list`, and `memory_delete` | `false` |

**Examples:**

```bash
export KDEPS_API_AUTH_TOKEN=dev-token      # Required when apiServer is configured
kdeps run workflow.yaml                    # Run from file
kdeps run ./my-agent/                      # Run from a directory containing workflow.yaml
kdeps run myapp.kdeps                      # Run from package
kdeps run workflow.yaml --dev              # Hot reload
kdeps run workflow.yaml --debug            # Debug logging
kdeps run workflow.yaml --port 16395       # Custom port
kdeps run workflow.yaml --interactive      # prompt alongside server
```

When you configure `apiServer`, kdeps does not start without `KDEPS_API_AUTH_TOKEN` or `api_auth_token` in `~/.kdeps/config.yaml`. Clients must send `Authorization: Bearer <token>` on workflow routes. See [Security reference](/reference/security).

---

## `kdeps [path]` (run a workflow)

`kdeps` with no command runs a workflow, as `kdeps run` does: the one at `path`, or `./workflow.yaml` in the current folder. With neither, it prints help.

```bash
kdeps                    # runs ./workflow.yaml
kdeps ./my-agent/        # runs that workflow
```

---

## `kdeps validate`

Validate workflow configuration against schema and business rules.

```bash
kdeps validate [workflow.yaml | directory] [flags]
```

**What it validates:**
- YAML syntax
- Schema compliance (JSON Schema)
- Resource dependencies
- Expression syntax
- Circular dependency detection
- Business rules
- Static analysis (unreachable resources, bad expression refs, missing component inputs)

**Examples:**

```bash
kdeps validate workflow.yaml
kdeps validate .                            # Validate all in directory
kdeps validate myapp.kdeps                  # Validate package
```

**Output:**
```
Validating: workflow.yaml

✓ YAML syntax valid
✓ Schema validation passed
✓ Resource dependencies resolved
✓ No circular dependencies
✓ Expression syntax valid

Workflow validated successfully
```

---

## `kdeps test`

Run the `evals:` cases from `workflow.yaml` against the router of the workflow, in the process. kdeps does not bind a port.

```bash
kdeps test [workflow.yaml | directory] [flags]
```

**Flags:**

| Flag | Description | Default |
|---|---|---|
| `--fail-fast` | Stop at the first failing case | `false` |

**Examples:**

```bash
export KDEPS_API_AUTH_TOKEN=dev-token   # Required when apiServer is configured
kdeps test ./my-agent/                   # Every case in workflow.yaml
kdeps test workflow.yaml                 # Single file
kdeps test myapp.kdeps                   # Packaged workflow
kdeps test . --fail-fast                 # Stop at the first failure
```

The request path is the same one a live client gets: middleware, bearer auth, route matching, executors, and `apiResponse` formatting. If a case sends no `Authorization` or `X-API-Key` header, kdeps sends it with `KDEPS_API_AUTH_TOKEN` as a bearer token. Setup matches `kdeps run`: `.j2` rendering, schema validation, environment setup, and the LLM backend when a chat resource needs one.

**Output:**

```text
Running 2 eval(s) for my-agent

  PASS  health (200, 0s)
  FAIL  chat: expected status 200, got 500
        status: 500
        body: {"success":false,"error":{"code":"INTERNAL_ERROR",...}}

1 passed, 1 failed
```

Exits `0` when every case passes and `1` when any case fails or the workflow defines no `evals:`. A failing case prints its status and the first 400 bytes of the response.

See [Test a workflow](/workflow/testing) for the `evals:` format and assertion reference.

---

## `kdeps new`

Create a new AI agent with interactive prompts.

```bash
kdeps new [agent-name] [flags]
```

**Flags:**

| Flag | Description | Default |
|---|---|---|
| `--template, -t` | Agent template to use | `api-service` |
| `--force` | Overwrite existing directory | `false` |

**Available templates:** `api-service`, `sql-agent`, `agency`, `whatsapp-bot` (a WhatsApp Cloud API webhook: verification handshake, signed messages answered by an LLM and sent back through the `http_request` component), `assistant` (a coding assistant: an `interactive` prompt and a `chat` with `loop:` over the file and shell tools)

**Examples:**

```bash
kdeps new my-agent                           # Uses api-service template
kdeps new my-agent --template sql-agent      # SQL agent template
kdeps new my-bot --template whatsapp-bot     # WhatsApp bot (webhook + Graph API reply)
kdeps new my-helper --template assistant     # coding assistant; then cd my-helper && kdeps
kdeps new my-agent --force                   # Overwrite existing
```

**Generated structure:**
```
my-agent/
├── workflow.yaml
├── resources/
│   ├── http_client.yaml
│   ├── llm.yaml
│   └── response.yaml
└── README.md
```

---

## `kdeps config`

Print the kdeps setup: version and platform, the config file, the LLM backend and models, the asset, model and registry folders, the asset profile, and each registry with its download time.

```bash
kdeps config          # print the setup
kdeps config --edit   # open ~/.kdeps/config.yaml in your editor, as kdeps edit
```

```text
kdeps:      3.5.0 (commit 1a2b3c4) darwin/arm64
Config:     /Users/me/.kdeps/config.yaml
Backend:    ollama
Models:     llama3.2
Assets:     /Users/me/.kdeps/assets
Profile:    agent
Models dir: /Users/me/.kdeps/models
Registries: /Users/me/.kdeps/registries
  kdeps/registry               downloaded 2026-11-23T10:00:00Z
  acme/tools                   not downloaded yet
```

---

## `kdeps edit`

Open the global kdeps configuration file (`~/.kdeps/config.yaml`) in your editor. Scaffolded if it does not exist.

```bash
kdeps edit [flags]
```

**Editor resolution:** `$KDEPS_EDITOR` > `$VISUAL` > `$EDITOR` > `vi`

```bash
kdeps edit
KDEPS_EDITOR=code kdeps edit                # Open in VS Code
```

---

## `kdeps doctor`

Run system health checks to diagnose common configuration and environment issues.

```bash
kdeps doctor [flags]
```

**Checks:**

| Check | Description |
|---|---|
| Config file | Existence of `~/.kdeps/config.yaml` |
| Config validation | Typos in API key names, missing keys |
| LLM backend | Models dir and cached llamafiles (file backend) or TCP connectivity (ollama) |
| Python | `python3` availability in PATH |
| Backend/API key | Cloud backend configured without its API key |
| Agents | Installed agent count |
| Env vars | Critical environment variables set |

Exits with code 1 when any check has FAIL status.

---

## `kdeps instrument`

List, show and remove sessions recorded with [`--instrument`](/reference/troubleshooting#record-a-session-for-a-model-to-debug).

```bash
kdeps run workflow.yaml --instrument=api-500
kdeps instrument list                   # saved sessions, newest first
kdeps instrument show billing-bug       # session.md and events.ndjson
kdeps instrument show billing-bug --trace   # also trace.log
kdeps instrument path billing-bug       # the folder
kdeps instrument rm billing-bug
```

The session folder is `~/.kdeps/instrument/<name>/`:

```text
session.md      # command, folder, start and end, event counts, how to read the rest
events.ndjson   # one JSON object per prompt, model call, tool call and workflow event
trace.log       # every log line at debug level, and the call-chain trace
```

Bare `--instrument` names the session by its start time. Put the name after `=`. With a space, kdeps uses the next argument (the workflow path) as the name.

---

## `kdeps --upgrade`

Check for a newer kdeps release and, for a standalone install, download, verify and install it. `--nightly` checks the nightly channel. `--target-version <v>` installs that exact version, also an older one.

```bash
kdeps --upgrade
```

**Flags:**

| Flag | Description | Default |
|---|---|---|
| `--nightly` | Check the nightly channel instead of the latest stable release | `false` |
| `--target-version` | Install this exact version instead of the latest, skipping the update check - can be older than the running build (a downgrade); takes priority over `--nightly` | `""` |

```bash
kdeps --upgrade --nightly
kdeps --upgrade --target-version 2.35.0
```

---

## Assets: `kdeps install`, `update`, `upgrade`, `uninstall`, `list`, `profile`

`kdeps install`, `kdeps upgrade`, and `kdeps uninstall` manage registry packages and the versioned assets of kdeps. The assets are harness, events, actions, presets, tool definitions, LLM server recipes, model registries, templates, schemas, and components. An argument that names an asset set (`tools`) or item (`tools/bash_exec`) is an asset. All other arguments are packages.

```bash
kdeps update                          # newest registries; list what kdeps upgrade would change, install nothing
kdeps upgrade                         # the chosen profile and everything installed since, to their newest versions
kdeps upgrade harness                 # every item in one set
kdeps install harness/safety          # one item at its newest version (unpins it), with what it requires
kdeps install harness/safety@1.2.0    # exactly that version, pinned (older = downgrade)
kdeps uninstall presets/frugal            # remove one item (restore with: kdeps install presets/frugal)
kdeps list --assets                   # every item, its version, state, shelf and ~context tokens
kdeps list rag                        # one shelf (model, context, harness, tools, memory, rag, agent, guardrails, evals, workflow, deploy)
kdeps profile agent                   # install a profile: minimum, workflows, agent, deploy, full (or sets and items: docker,tools/bash_exec)
kdeps profile                         # choose the profile again, or tick sets and single items yourself (custom); needs a terminal
```

| Command | Assets |
|---|---|
| `kdeps install <ref>...` | Install sets or items (`@version` pins), and what they require; each line shows the item's estimated LLM context tokens, and the total is printed last |
| `kdeps update` | Download every registry again and list the assets that have a newer version; installs nothing |
| `kdeps upgrade [ref]...` | No args: every installed asset and the profile; a package ref reinstalls it at its formula's version |
| `kdeps uninstall <ref>...` | Remove items (comma-separated or several args); required items cannot be removed |
| `kdeps list --assets` | Every asset with its version, state (`source`, `downloaded`, `pinned`, `removed`), shelf and estimated LLM context tokens, with the installed total |
| `kdeps list <shelf>` | Assets on one shelf. Item ids stay `set/name` |
| `kdeps profile [name]` | Install a profile (`minimum`, `workflows`, `agent`, `deploy`, `full`, `custom`, or a comma-separated list of sets and `set/name` items) and record it as what `upgrade` keeps current; no name opens the picker |

[Installation](/start/installation) lists the profiles and what each profile installs.

## `kdeps llamafile`

Manage the local model registry. Covers llamafile, GGUF (llama.cpp), and Ollama models. Aliases like `llama3.2:1b` resolve to cached model files in `~/.kdeps/models/`. kdeps downloads models on first use with aria2c (fast parallel downloads with resume) or built-in HTTP. Ctrl+C cancels a download immediately.

An interrupted aria2c download stops. It does not start again through the HTTP fallback. kdeps keeps the partial file, so the next attempt continues from it.

```bash
kdeps llamafile list      # all known aliases (LF + GGUF + Ollama) with size, quant, and URL
kdeps upgrade models       # refresh the registries (llamafile + GGUF) from kdeps.io
```

`list` shows a TYPE column (LF, GGUF, or model name) for each entry. The registries are the `models/llamafile` and `models/gguf` assets. A job collects them each night from HuggingFace and publishes them as new versions. Install them with `kdeps upgrade models`. Models that you register yourself (HuggingFace search or a custom URL) are in `~/.kdeps/llamafile_versions.yaml` and `~/.kdeps/gguf_versions.yaml`. kdeps adds them on top.

kdeps finds Ollama models in the local `ollama list` output.

The free text models of OpenRouter are a separate registry, `models/openrouter`. `tools/openrouter-harvester/harvest.py` (`make harvest-openrouter`) collects them each day from the public models API. Install them with `kdeps upgrade models/openrouter`.

---

## `kdeps knowledge`

Manages the [knowledge vault](/workflow/agent-loop/knowledge): markdown notes that a workflow `chat` recalls on each call, with no full read. On by default in `~/.kdeps/notes`, set up by the first chat call. The structural graph refreshes before a chat call's recall, on a vault search with `graphBoost`, and on skill linking. `kdeps knowledge build` runs that build by hand. `--extract` adds the model step and is off unless set.

```bash
kdeps knowledge init [folder]       # scaffold projects/, shared/, inbox/, archive/, KDEPS.md (default ~/.kdeps/notes)
kdeps knowledge build               # graph under ~/.kdeps/knowledge/<vault-id>; changed notes only
kdeps knowledge build --extract     # the subagent model reads changed notes first: inferred edges with confidence
kdeps knowledge build --export      # also copy GRAPH_REPORT.md and graph.html into the vault
kdeps knowledge check               # structural one-home: duplicate summaries, canonical ids, dangling links
kdeps knowledge recall "why monthly billing"   # what recall would give a chat, with the relation chain
kdeps knowledge explain "billing"   # matching edges, with the file and line each one came from
kdeps knowledge report              # print GRAPH_REPORT.md (deterministic communities)
```

kdeps writes the graph, `GRAPH_REPORT.md`, and `graph.html` under `~/.kdeps/knowledge/<vault-id>/`. `<vault-id>` is a hash of the vault path.

| Flag | Description | Default |
|---|---|---|
| `--vault` | Vault folder | `knowledge.vault` in `~/.kdeps/config.yaml`, else `~/.kdeps/notes` |
| `--extract` (build) | Have the subagent model read changed notes first | `false` |
| `--export` (build) | Copy the report and the explorer page into the vault | `false` |
| `--model` (build) | Model for `--extract` | `llm.subagent` when it names a model, else `knowledge.extract.model`, else the current model |
| `--limit` (build) | Most notes `--extract` reads | `0` (all changed) |

`check` exits non-zero when it reports anything. It reports structural duplicates. It does not decide that two different wordings are the same fact.

---

## See also

- [CLI overview](/reference/cli) - global flags, exit codes, env vars, workflows
- [Registry commands](/registry/cli) - search, install, publish
- [Packaging commands](/deploy/cli) - bundle, export, build
