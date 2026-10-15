# Dev commands

Commands for local development: run, agent REPL, validate, scaffold, configure, and diagnose.

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
| `--interactive` | Open an interactive LLM REPL alongside the running workflow | `false` |
| `--file` | File path to process (file input source only). Takes priority over stdin, `KDEPS_FILE_PATH`, and `input.file.path` config | |
| `--events` | Emit structured NDJSON execution events to stderr (resource lifecycle, failure classification) | `false` |
| `--memory` | Enable the agent memory store in workflow mode - resources can use `memory_save`/`memory_search`/`memory_list`/`memory_delete` expression functions | `false` |

**Examples:**

```bash
export KDEPS_API_AUTH_TOKEN=dev-token      # Required when apiServer is configured
kdeps run workflow.yaml                    # Run from file
kdeps run ./my-agent/                      # Run from a directory containing workflow.yaml
kdeps run myapp.kdeps                      # Run from package
kdeps run workflow.yaml --dev              # Hot reload
kdeps run workflow.yaml --debug            # Debug logging
kdeps run workflow.yaml --port 16395       # Custom port
kdeps run workflow.yaml --interactive      # LLM REPL alongside server
```

When `apiServer` is configured, kdeps refuses to start without `KDEPS_API_AUTH_TOKEN` or `api_auth_token` in `~/.kdeps/config.yaml`. Clients must send `Authorization: Bearer <token>` on workflow routes. See [Security reference](/reference/security).

---

## `kdeps [path]` (agent REPL)

Start agent mode - an interactive LLM loop where whole workflows and components are registered as callable tools. Pass a path to expose workflows as tools, or omit it for a bare REPL with no workflow tools.

```bash
kdeps [path] [flags]
```

**Flags:**

| Flag | Default | Description |
|---|---|---|
| `--model` | `KDEPS_AGENT_MODEL` or `llama3.2:1b` | LLM model name |
| `--backend` | `KDEPS_AGENT_BACKEND` or `file` | LLM backend (`file` = local llamafile, no server) |
| `--base-url` | `KDEPS_AGENT_BASE_URL` | LLM API base URL |
| `--system` | (none) | System prompt injected at conversation start |
| `--skill` | (none) | Load a skill file or directory (repeatable) |
| `--resume` | (none) | Resume a saved session by ID |
| `--debug` | false | Enable debug logging |

**Examples:**

```bash
kdeps                                                         # bare REPL, no workflow tools
kdeps ./my-agent/                                             # one tool: metadata.name
kdeps ./agents/                                               # all workflows in folder as tools
kdeps ./my-agent/ --model mistral
kdeps ./agents/ --system "You are a helpful assistant."
```

See [Agent Mode](/agent/) for full details.

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

**Available templates:** `api-service`, `sql-agent`, `agency`, `whatsapp-bot` (a WhatsApp Cloud API webhook: verification handshake, signed messages answered by an LLM and sent back with `httpClient`)

**Examples:**

```bash
kdeps new my-agent                           # Uses api-service template
kdeps new my-agent --template sql-agent      # SQL agent template
kdeps new my-bot --template whatsapp-bot     # WhatsApp bot (webhook + Graph API reply)
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

## `kdeps --upgrade`

Check for a newer kdeps release and, for a standalone install, download/verify/install it. Same flow as the REPL's `/upgrade` command - see [Agent loop REPL features - Updating kdeps](/agent/repl-updating#updating-kdeps) for the full behavior (what each install method does, checksum verification, etc.).

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

## Assets: `kdeps install`, `update`, `uninstall`, `list`, `profile`

`kdeps install`, `kdeps update` and `kdeps uninstall` install, update and remove kdeps' versioned assets - harness sections, events, actions, presets, themes, tool definitions, slash commands, LLM server recipes, model registries, project templates, schemas and components - as well as registry packages. An argument naming an asset set (`tools`) or item (`tools/bash_exec`) is an asset; anything else is a package. Same as the REPL's `/update`; see [Updating assets](/agent/repl-updating#updating-assets) for how versions, pins and removal work.

```bash
kdeps update                          # the chosen profile and everything installed since, to their newest versions
kdeps update --check                  # list available updates, change nothing
kdeps update harness                  # every item in one set
kdeps install harness/safety          # one item at its newest version (unpins it), with what it requires
kdeps install harness/safety@1.2.0    # exactly that version, pinned (older = downgrade)
kdeps uninstall themes/vim themes/linux   # remove items (restore with: kdeps install themes/vim)
kdeps list --assets                   # every item, its version, state and ~context tokens
kdeps profile agent                   # install a profile: minimum, workflows, agent, deploy, full (or sets and items: docker,tools/bash_exec)
kdeps profile                         # choose the profile again, or tick sets and single items yourself (custom); needs a terminal
```

| Command | Assets |
|---|---|
| `kdeps install <ref>...` | Install sets or items (`@version` pins), and what they require; each line shows the item's estimated LLM context tokens, and the total is printed last |
| `kdeps update [ref]...` | No args: every installed asset and the profile; `--check` lists updates without installing |
| `kdeps uninstall <ref>...` | Remove items (comma-separated or several args); required items cannot be removed |
| `kdeps list --assets` | Every asset with its version, state (`source`, `downloaded`, `pinned`, `removed`) and estimated LLM context tokens, with the installed total |
| `kdeps profile [name]` | Install a profile (`minimum`, `workflows`, `agent`, `deploy`, `full`, `custom`, or a comma-separated list of sets and `set/name` items) and record it as what `update` keeps current; no name opens the picker |

The profiles and what each installs are listed in [Updating assets](/agent/repl-updating#updating-assets).

---

## `kdeps chat`

Interactive AI assistant that generates and runs kdeps workflows from natural language.

```bash
kdeps chat [flags]
```

**Flags:**

| Flag | Description | Default |
|---|---|---|
| `--model` | LLM model for workflow generation | From config |
| `--base-url` | LLM backend base URL | `http://localhost:11434` |
| `--session` | Resume a previous session by ID | New session |
| `--no-execute` | Generate workflow but do not allow `/run` | `false` |

**REPL slash commands:**

| Command | Description |
|---|---|
| `/show` | Print the generated workflow YAML |
| `/run` | Execute the workflow |
| `/save [path]` | Save workflow to directory |
| `/export` | Show Kubernetes manifests |
| `/reset` | Clear conversation and start fresh |
| `/quit` | Exit |

```bash
kdeps chat                                    # Start interactive assistant
kdeps chat --model gpt-4o                     # Use specific model
echo "list files in /tmp" | kdeps chat --no-execute
```

## `kdeps llamafile`

Manage the local model registry. Covers llamafile, GGUF (llama.cpp), and Ollama models. Aliases like `llama3.2:1b` resolve to cached model files in `~/.kdeps/models/`; models are downloaded on first use with aria2c (fast parallel downloads with resume) or built-in HTTP. Ctrl+C cancels an in-flight download immediately - an interrupted aria2c download aborts instead of restarting via the HTTP fallback, and the partial file is kept so the next attempt resumes.

When [llmfit](https://github.com/AlexsJones/llmfit) is installed, the agent REPL's `/model` picker shows a hardware-fit score (0-100) and level for each local model - `Perfect`, `Good`, `Marginal`, or `Too Tight` - based on your machine's RAM/VRAM, so you can tell before downloading whether a model will actually run. Fit levels are searchable in the picker: type `perfect` or `tight` to filter. When aria2c or llmfit is not installed, the REPL prints a one-line install tip at startup.

```bash
kdeps llamafile list      # all known aliases (LF + GGUF + Ollama) with size, quant, and URL
kdeps update models       # refresh the registries (llamafile + GGUF) from kdeps.io
```

`list` shows a TYPE column (LF, GGUF, or model name) for each entry. The registries are the `models/llamafile` and `models/gguf` assets: harvested nightly from HuggingFace, published as new versions, and installed with `kdeps update models` (see [Updating assets](/agent/repl-updating#updating-assets)). Models you register yourself (HuggingFace search or a custom URL) live in `~/.kdeps/llamafile_versions.yaml` and `~/.kdeps/gguf_versions.yaml` and are merged on top. Ollama models are discovered from the local `ollama list` output.

---

## `kdeps knowledge`

Sets up and maintains the [knowledge vault](/agent/knowledge): the markdown notes the agent recalls from each turn without reading them. It is on by default in `~/.kdeps/notes` and set up by the first agent session.

```bash
kdeps knowledge init [folder]       # scaffold KDEPS.md, schema.md, records/, PARA folders (default ~/.kdeps/notes)
kdeps knowledge build               # build the link graph (only changed notes and code files are re-read)
kdeps knowledge check               # one home per fact: next steps outside cards, duplicate summaries, dangling links
kdeps knowledge recall "why monthly billing"   # what recall would hand the agent
kdeps knowledge report              # GRAPH_REPORT.md: god nodes, Louvain communities, surprising connections
```

| Flag | Description | Default |
|---|---|---|
| `--vault` | Vault folder | `knowledge.vault` in `~/.kdeps/config.yaml`, else `~/.kdeps/notes` |

`check` exits non-zero when it reports anything.

---

## `kdeps konfig export`

Export the current effective agent-loop config - tuning, harness, themes, and skills - to one self-contained YAML file. See [konfig](/agent/konfig) for the full file format and the REPL's `/konfig export` equivalent.

```bash
kdeps konfig export              # writes ./konfig.yaml
kdeps konfig export path/to/mine.yaml
```

Reads whatever is persisted in `~/.kdeps/agent-loop-settings.yaml`, or built-in defaults if nothing has ever been customized - so this works even against a completely fresh `~/.kdeps`.

---

## See also

- [CLI overview](/reference/cli) - global flags, exit codes, env vars, workflows
- [Registry commands](/registry/cli) - search, install, publish
- [Packaging commands](/deploy/cli) - bundle, export, build
