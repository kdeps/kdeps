# Dev Commands

Commands for local development: run, serve, validate, scaffold, configure, and diagnose.

## `kdeps run`

Run a workflow locally (default execution mode).

```bash
kdeps run [workflow.yaml | package.kdeps] [flags]
```

**Arguments:**
- `workflow.yaml` -- Path to workflow file or directory containing `workflow.yaml`
- `package.kdeps` -- Path to packaged workflow file

**Flags:**

| Flag | Description | Default |
|---|---|---|
| `--dev` | Enable hot reload mode | `false` |
| `--port` | API server port number | From workflow config |
| `--debug` | Enable debug logging | `false` |
| `--interactive` | Open an interactive LLM REPL alongside the running workflow | `false` |
| `--self-test` | Run `tests:` block after server starts, keep running | `false` |
| `--self-test-only` | Run `tests:` block then exit (non-zero on failure) | `false` |
| `--write-tests` | Generate tests from resources and write to workflow file, then exit | `false` |

**Examples:**

```bash
kdeps run workflow.yaml                    # Run from file
kdeps run myapp.kdeps                      # Run from package
kdeps run workflow.yaml --dev              # Hot reload
kdeps run workflow.yaml --debug            # Debug logging
kdeps run workflow.yaml --port 16395       # Custom port
kdeps run workflow.yaml --write-tests      # Generate test scaffold
kdeps run workflow.yaml --self-test        # Run tests, keep server
kdeps run workflow.yaml --self-test-only   # CI: run tests, exit
kdeps run workflow.yaml --interactive      # LLM REPL alongside server
```

**Self-test workflow:**

```bash
# Step 1: scaffold tests from your workflow resources
kdeps run workflow.yaml --write-tests

# Step 2: review and edit workflow.yaml tests: section

# Step 3: run them in CI
kdeps run workflow.yaml --self-test-only
echo "Exit code: $?"
```

When no explicit `tests:` block is present, `--self-test` and `--self-test-only` auto-generate smoke tests from workflow routes and resources at runtime (nothing is written to disk).

---

## `kdeps serve`

Start agent mode -- an interactive LLM loop where whole workflows and components are registered as callable tools. Pass a single file to expose one workflow tool plus its components, or a folder to expose every workflow and agency inside as separate tools along with all their components.

```bash
kdeps serve <path> [flags]
```

**Flags:**

| Flag | Default | Description |
|---|---|---|
| `--model` | `KDEPS_AGENT_MODEL` or `llama3.2` | LLM model name |
| `--backend` | `KDEPS_AGENT_BACKEND` or `ollama` | LLM backend |
| `--base-url` | `KDEPS_AGENT_BASE_URL` | LLM API base URL |
| `--system` | (none) | System prompt injected at conversation start |
| `--debug` | false | Enable debug logging |

**Examples:**

```bash
kdeps serve workflow.yaml
kdeps serve workflow.yaml --model mistral
kdeps serve workflow.yaml --system "You are a helpful assistant."
```

See [Agent Mode](/modes/agent-mode) for full details.

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
kdeps validate workflow.yaml --verbose
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
| `--template, -t` | Agent template to use | Interactive selection |
| `--force` | Overwrite existing directory | `false` |

**Available templates:** `api-service`, `sql-agent`, `file-processor`, `cli-tool`

**Examples:**

```bash
kdeps new my-agent                           # Interactive mode
kdeps new my-agent --template api-service    # Quick start
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
├── .env.example
└── README.md
```

---

## `kdeps edit`

Open the global kdeps configuration file (`~/.kdeps/config.yaml`) in your editor. Scaffolded if it doesn't exist.

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
| Ollama | TCP connectivity to the Ollama server |
| Python | `python3` availability in PATH |
| Backend/API key | Cloud backend configured without its API key |
| Agents | Installed agent count |
| Env vars | Critical environment variables set |

Exits with code 1 when any check has FAIL status.

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
| `--api-key` | API key for online LLM providers | From env |
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

## See Also

- [CLI Overview](/reference/cli/) -- global flags, exit codes, env vars, workflows
- [Registry Commands](/reference/cli/registry) -- search, install, publish
- [Packaging Commands](/reference/cli/packaging) -- bundle, export, build
