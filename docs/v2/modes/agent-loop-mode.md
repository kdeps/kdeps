# Agent mode

Agent mode starts an interactive LLM REPL where whole workflows and components are registered as callable tools. The LLM decides which tool to invoke based on the user's prompt. Workflow tools run the full pipeline atomically so all `requires:` dependencies resolve correctly. For the deterministic request/response pipeline instead, see [Workflow mode](/modes/workflow-mode).

Running `kdeps` with no arguments starts a bare REPL with no workflow tools - built-in tools (web_search, bash_exec, file ops, memory, etc.) are still available. Pass a path to also load workflows and agencies as tools.

## Starting the agent loop

```bash
kdeps                              # runs the agent loop REPL
kdeps --model llama3.2 --system "You are a DevOps assistant."  # override model/system prompt
kdeps --skill ~/.kdeps/skills/     # load skill files
kdeps --resume <session-id>        # continue a saved session
```

## Single workflow vs folder

```bash
kdeps ./my-agent/     # registers the workflow as an LLM-callable tool (named after metadata.name)
kdeps ./agents/       # registers every workflow and agency in the folder as a separate tool
```

When you point at a folder, kdeps discovers every workflow and agency file inside it (recursively). Each becomes a separate tool. The tool name is `metadata.name` from the workflow's manifest - not the filename.

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

Given a workflow whose `metadata.name` is `my-agent`, running `kdeps ./my-agent/` gives the LLM one tool named `my-agent`. When it calls that tool, kdeps runs the full workflow DAG - every resource in dependency order - and returns `apiResponse.response` to the LLM. The LLM can then call more tools or produce a final answer.

## Command and flags

```bash
kdeps [path] [flags]
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
| `--resume` | (none) | Session ID to resume a previous conversation |
| `--stealth` | false | Muted UI - dark gray, model name barely visible (for use in public) |
| `--debug` | false | Enable debug logging |

```bash
# Environment variables (flags override these)
KDEPS_AGENT_MODEL=llama3.2
KDEPS_AGENT_BACKEND=file              # default: local llamafile
KDEPS_AGENT_BASE_URL=http://localhost:11434
KDEPS_STEALTH=1                       # same as --stealth (1, true, or yes)
```

## Examples

```bash
kdeps                                            # bare REPL, built-in tools only
kdeps ./my-agent/                                # register one workflow as a tool
kdeps ./agents/                                  # register every workflow in the folder
kdeps ./agents/ --model mistral --system "You are a data analyst."
kdeps --backend gguf --model qwen3.5-4b          # local GGUF model
KDEPS_AGENT_BACKEND=openai kdeps ./agents/ --model gpt-4o
kdeps --resume abc123def456                      # resume a session
```

## More on the agent loop

| Topic | Page |
|-------|------|
| Slash commands and auto-detected shell/file mentions | [REPL slash commands](/modes/agent-loop-commands) |
| Built-in tool catalog, permission modes, lean mode | [Built-in tools](/modes/agent-loop-tools) |
| Shell execution (`!cmd`, Ctrl+C / Ctrl+Z, jobs) | [Shell execution](/modes/agent-loop-shell) |
| Live status line and stall detection during a tool run | [Tool execution monitoring](/modes/agent-loop-monitoring) |
| Task decomposition and forward-only goal enforcement | [Goal-directed execution](/modes/agent-loop-goals) |
| Independent review of the final answer | [Judge panel](/modes/agent-loop-judges) |
| `/model`, `/context`, auto-routing, running local servers | [Local model management](/modes/agent-loop-models) |
| Task / team / cron registries (`task_*`, `team_*`, `cron_*`) | [Agent registries](/modes/agent-loop-registries) |
| One-time permission exceptions for a denied tool call | [Approval tokens](/modes/agent-loop-approvals) |
| Optional `turo` token reducer | [Prompt reduction (turo)](/modes/agent-loop-turo) |
| Skills, prompt templates, `KDEPS.md` instructions | [Skills and prompt templates](/modes/agent-loop-skills) |
| Pasting, rendering, stealth, sessions, notifications, updates | [Agent loop REPL features](/modes/agent-loop-repl) |

## Differences from workflow mode

| | Workflow mode (`kdeps run`) | Agent mode (`kdeps [path]`) |
|--|-----------------------------|---------------------------------|
| Execution | DAG, deterministic | LLM loop, tool-driven |
| Entry point | `metadata.targetActionId` | User prompt |
| Unit of work | Individual resources | Whole workflows |
| Tools exposed | Functions in `chat.tools` | One per workflow + one per component |
| Input | Single workflow path | Optional file or folder |
| Session memory | None | Multi-turn, persistent JSONL |

## See also

- [Agent loop REPL features](/modes/agent-loop-repl) - pasting, rendering, sessions, updates
- [Skills and prompt templates](/modes/agent-loop-skills) - context files that teach the agent
- [REPL slash commands](/modes/agent-loop-commands) - full command reference
- [Workflow mode](/modes/workflow-mode) - deterministic DAG pipelines
- [LLM provider reference](/reference/llm-providers) - backend config and model names
- [Agencies](/concepts/agency) - multi-agent orchestration
