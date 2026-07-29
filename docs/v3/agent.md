# Coding agent

Primary mode: a **CLI coding agent** in your terminal. Plans tasks, calls tools, finishes the goal.

```bash
kdeps                              # model-only REPL
kdeps .                            # this project — workflows become tools
kdeps ./my-agent/                  # one workflow = one tool
kdeps ./agents/                    # every workflow under path = a tool
kdeps --model deepseek-v4-flash --system "You are a DevOps assistant."
kdeps --resume <session-id>
kdeps --skill ~/.kdeps/skills/
```

No YAML required to chat. Add a path when you want project tools.

## How tools register

1. Discover `workflow.yaml` under the path  
2. Register each as a tool using `metadata.name` + `metadata.description`  
3. Start the LLM loop  

The model sees **tool name, description, and input schema** — not individual resources. The DAG inside is an implementation detail.

```yaml
# workflow.yaml
metadata:
  name: web-researcher          # tool id (lowercase, hyphenated)
  description: "Fetch a URL and answer questions about it"
```

**Names and descriptions matter.** Overlapping tools confuse the model.

### Tool input / output

Calls pass an `input` field. Inside the workflow:

```yaml
prompt: "Process: {{ get('input') }}"
```

Return value is the terminal `apiResponse` body (what `targetActionId` produces).

### Folder of specialists

```text
agents/
  research/workflow.yaml    # metadata.name: research-agent
  writer/workflow.yaml      # metadata.name: writer-agent
```

```bash
kdeps ./agents/    # two tools; model composes them
```

## Built-in tools (high signal)

Always available in the loop (subset — type `/help` / watch tool list for the full set):

| Group | Examples |
|-------|----------|
| Web | `web_search`, `wikipedia`, `web_scraper`, `http_request` |
| Files | `read_file`, `write_file`, `edit_file`, `list_files`, `search_local` |
| Code | `code_search`, `code_definition`, `code_references`, `code_diagnostics`, … |
| Shell | `bash_exec`, `bash_job_list`, `bash_job_wait` |
| Memory | `memory_save`, `memory_search`, `memory_list`, `memory_delete` |
| Data | `sql_query`, `sql_list_tables`, `calculator` |
| Docs / RAG | `load_document`, `embedding_*`, `retrieve_context` |
| Orchestration | `task_*`, `team_*`, `cron_*`, `approval_*` |

**bash:** Ctrl+C cancels and returns partial output; Ctrl+Z backgrounds (`bash_job_wait` later).  
**rtk:** if `rtk` is on `PATH`, `bash_exec` compresses output automatically.  
**Git:** agent commits can add a `Co-Authored-By: kdeps (…)` trailer.

## Permissions and presets

```bash
export KDEPS_PERMISSION_MODE=read-only          # or workspace-write, danger-full-access (default)
export KDEPS_LEAN_MODE=true                     # no bash / network tools
export KDEPS_AGENT_PRESET=audit|explain|implement
export KDEPS_ALLOW_BASH=false
export KDEPS_BASH_MODE=read-only
```

| Preset | Permission | Tools |
|--------|------------|--------|
| `audit` / `explain` | ReadOnly | Lean |
| `implement` | WorkspaceWrite | Lean + file writes |

Blocked calls can request a one-shot override via `approval_request` / `approval_grant` (REPL tools).

## Goals

```text
prompt -> task list -> [task 1] -> [task 2] -> ... -> answer
```

Advance with `task_complete` / `task_fail`. Steer with `/goal`.

## Slash commands

Type `/help` for the live list.

| Command | Purpose |
|---------|---------|
| `/model [name]` | Show or switch model |
| `/model list` / `ps` | Models / local servers |
| `/model tool set …` | Persist loop settings |
| `/clear` `/compact` | History |
| `/session …` | Save / load |
| `/thinking …` | Extended reasoning |
| `/turo …` | Token reducer if installed |
| `/goal …` | Task plan |
| `/skills` | Loaded skills |
| `! <cmd>` / `!! <cmd>` | Shell with / without model turn |
| `/exit` | Leave |

## Skills and settings

```bash
kdeps --skill ~/.kdeps/skills/
```

Loop settings: `~/.kdeps/agent-loop-settings.yaml` (or `/model tool set …`).

## Workflow next

Fixed APIs and DAGs: [Workflow mode](/workflow). Multi-agent packages: [Agencies](/agencies). Map: [CLI](/cli).
