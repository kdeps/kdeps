# kdeps agent

The kdeps agent is the agent loop of a workflow: a `chat` with [`loop:`](/workflow/resources/llm#agent-loop) that calls tools over many rounds, with persistent memory, skills, and an opt-in judge panel (the [eval](/reference/glossary#eval)). `kdeps --agent-loop` opens it as a REPL on your machine. The default model is local. It needs no API key and runs fully offline.

To send prompts to an [LLM server](/llm-server/) that you run, set `base_url` under `llm:`. Workflows and components register as tools. The model selects the tools to call.

```bash
kdeps --agent-loop   # bare REPL - built-in tools only
kdeps --agent-loop ./my-agent # also load that agent's workflows/components as tools
```

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1   # your LLM server, your network
  models:
    - llama3.2                            # one model, or several
```

The boundary rules are on [Data sovereignty](/start/data-sovereignty). The server page has the [client contract](/llm-server/#client-contract).

The same `workflow.yaml` is the pipeline. `kdeps` runs it. This page is the REPL for a `chat` with `loop:`. To run several workflows as one system, see [Agencies](/agencies/).

---

## Single workflow vs folder

When you point at a folder, kdeps discovers every workflow and agency file inside it (recursively). Each becomes a separate tool. The tool name is the workflow's `name` under `metadata:`, not the filename.

| Target | Workflow/agency/component tools registered |
|--------|-----------------|
| No path | None (built-in tools only) |
| Single workflow file/dir | One tool (`metadata.name`) + one per component |
| Single agency file | One tool (`agency metadata.name`) |
| Folder | One tool per workflow/agency found recursively + component tools |

## How it works

```d2
direction: down

A: user prompt {shape: oval}
B: "LLM receives prompt\ntool registry: one tool per workflow, one per agency, one per component"
C: tool type? {shape: diamond}
D: "kdeps runs full workflow pipeline\nall requires: deps resolve in order"
E: "kdeps runs agency entry-point pipeline\ninternal agents resolve via agent: resource type"
H: "kdeps runs component in isolation\ninputs map to component interface fields"
F: more tools needed? {shape: diamond}
G: final answer {shape: oval}

A -> B
B -> C: LLM picks a tool
C -> D: workflow
C -> E: agency
C -> H: component
D -> F: apiResponse returned to LLM
E -> F: result returned to LLM
H -> F: result returned to LLM
F -> C: yes
F -> G: no
```

Given a workflow whose `metadata.name` is `my-agent`, running `kdeps --agent-loop ./my-agent/` gives the LLM one tool named `my-agent`. When it calls that tool, kdeps runs the full workflow DAG - every resource in dependency order - and returns `apiResponse.response` to the LLM. The LLM can then call more tools or produce a final answer.

## Command and flags

```bash
kdeps --agent-loop [path] [flags]
```

`[path]` is optional. When provided it must be a workflow or agency file or directory.

| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `KDEPS_AGENT_MODEL` or `llama3.2:1b` | LLM model name |
| `--backend` | `KDEPS_AGENT_BACKEND` or `file` | LLM backend (`file`, `gguf`, `ollama`, `openai`, ...) |
| `--base-url` | `KDEPS_AGENT_BASE_URL` | LLM API base URL |
| `--system` | (none) | System prompt injected at conversation start |
| `--skill` | (none) | Path to a skill file or directory (repeatable) |
| `--prompt` | (none) | Path to a prompt templates directory (repeatable) |
| `--resume` | (none) | Session ID to resume directly (skips the picker) |
| `--new` | false | Start clean, skipping this folder's resume picker |
| `--index` | (none) | Search a folder outside the project with it (repeatable): added to the card and indexed first. See [search folders](/agent/knowledge#search-folders) |
| `--theme` | `normal` | REPL palette. Only `normal` applies. |
| `--debug` | false | Enable debug logging |

```bash
# Environment variables (flags override these)
KDEPS_AGENT_MODEL=llama3.2:1b
KDEPS_AGENT_BACKEND=file              # default: local llamafile
KDEPS_AGENT_BASE_URL=http://localhost:11434/v1
KDEPS_THEME=normal                    # only normal applies
```

### Settings from the workflow

When the workflow (the one at `[path]`, or `./workflow.yaml`) targets a `chat` with [`loop:`](/workflow/resources/llm#agent-loop), that chat defines the session:

```yaml
# resources/assistant.yaml
chat:
  model:
    name: qwen2.5:7b                 # the session's model; --model / --backend still win
  loop:
    tools: [read_file, edit_file]    # the only built-in tools offered; project tools stay
    maxRounds: 25                    # tool rounds per turn
```

A workflow that does not parse, or names an unknown tool, is reported and the session keeps its own settings.

## Examples

```bash
kdeps --agent-loop                               # bare REPL, built-in tools only
kdeps --agent-loop ./my-agent/                                # register one workflow as a tool
kdeps --agent-loop ./agents/                                  # register every workflow in the folder
kdeps --agent-loop ./agents/ --model mistral --system "You are a data analyst."
kdeps --agent-loop --backend gguf --model qwen3.5-4b          # local GGUF model
KDEPS_AGENT_BACKEND=openai kdeps --agent-loop ./agents/ --model gpt-4o
kdeps --agent-loop --resume abc123def456                      # resume a session
```

## More on the agent loop

| Topic | Page |
|-------|------|
| Slash commands and auto-detected shell/file mentions | [REPL slash commands](/agent/commands) |
| Built-in tool catalog, permission modes, lean mode | [Built-in tools](/agent/tools) |
| Shell execution (`!cmd`, Ctrl+C / Ctrl+Z, jobs) | [Shell execution](/agent/shell) |
| Live status line and stall detection during a tool run | [Tool execution monitoring](/agent/monitoring) |
| Subagents: the `subagent` tool and knowledge extraction | [Subagents](/agent/subagent) |
| Review of a workflow chat answer | [Judge panel](/agent/judges) |
| `/model`, `/context`, auto-routing, running local servers | [Local model management](/agent/models) |
| One-time permission exceptions for a denied tool call | [Approval tokens](/agent/approvals) |
| Skills, prompt templates, `KDEPS.md` instructions | [Skills and prompt templates](/agent/skills) |
| Per-folder session history and the startup resume picker | [Sessions](/agent/repl-sessions#sessions) |
| Pasting, rendering, themes, notifications, updates | [Agent loop REPL features](/agent/repl) |

## The same file

`kdeps` runs `./workflow.yaml` as an HTTP API, a bot, a file, or an interactive prompt. `kdeps --agent-loop` opens a `chat` with `loop:` as this REPL. The steps around that chat stay in `requires:` order.

## See also

- [Agent loop REPL features](/agent/repl) - pasting, rendering, sessions, updates
- [Skills and prompt templates](/agent/skills) - context files that teach the agent
- [REPL slash commands](/agent/commands) - full command reference
- [Workflow](/workflow/) - the YAML pipeline
- [Input sources](/workflow/input-sources) - API, bot, file, or interactive prompt
- [LLM provider reference](/llm-server/providers) - backend config and model names
- [Agencies](/agencies/) - multi-agent orchestration
