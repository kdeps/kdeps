# Coding agent

Primary mode: a **CLI coding agent** in your terminal. Plans tasks, calls tools, finishes the goal.

```bash
kdeps                              # model-only REPL
kdeps .                            # this repo / project as tools
kdeps ./my-agent/                  # one workflow = one tool
kdeps ./agents/                    # every workflow under the path = a tool
kdeps --model deepseek-v4-flash --system "You are a DevOps assistant."
kdeps --resume <session-id>
kdeps --skill ~/.kdeps/skills/
```

No workflow files required to chat. Add a path when you want project tools.

## What it is good for

- Code and ops work in a real shell context
- Multi-step tasks with a plan (`/goal`)
- Calling your kdeps workflows as tools without leaving the REPL
- Local models (llamafile / GGUF) or cloud keys

## Start

```bash
kdeps --version   # expect 2.1.x on current releases
kdeps
```

First run may ask for a backend. **llamafile** needs no API key; model lands in `~/.kdeps/models/`.

Cloud:

```bash
export ANTHROPIC_API_KEY=...
export OPENAI_API_KEY=...
export DEEPSEEK_API_KEY=...
# or llm: in ~/.kdeps/config.yaml
```

## Root flags (agent)

| Flag | Purpose |
|------|---------|
| `[path]` | Load workflows / agencies under path as tools |
| `--model` | Model id (else env / config / default) |
| `--backend` | Backend id |
| `--base-url` | OpenAI-compat base URL |
| `--system` | System prompt every turn |
| `--resume` | Resume session id |
| `--skill` | Skill file or directory (repeatable) |
| `--debug` / `--verbose` | Logging |

## Tools

Built-ins always available in the loop (examples): web search, scraper, calculator, SQL helpers, bash, file ops.

Pass a path: each workflow (and agency) registers as a tool. When the model calls one, the **full workflow engine** runs — same DAG as `kdeps run`.

## Goals

```text
prompt -> task list -> [task 1] -> [task 2] -> ... -> answer
```

The model advances with `task_complete` / `task_fail`. Steer with `/goal`.

## Slash commands

Type `/help` for the live list. Common:

| Command | Purpose |
|---------|---------|
| `/help` | All commands |
| `/model [name]` | Show or switch model |
| `/model list` | Available models |
| `/model ps` | Running local servers |
| `/model tool set …` | Persist loop settings |
| `/clear` | Summarize and clear |
| `/compact` | Compact history |
| `/session …` | Save / load / list |
| `/thinking …` | Extended reasoning (when supported) |
| `/turo …` | Token reducer if `turo` on `PATH` |
| `/goal …` | Inspect or change plan |
| `/skills` | Loaded skills |
| `/exit` | Leave REPL |
| `! <cmd>` | Shell; model sees output |
| `!! <cmd>` | Shell; no model turn |

## Skills

```bash
kdeps --skill ~/.kdeps/skills/
```

Skills teach domain behavior (including scaffolding kdeps projects). List with `/skills`.

## Settings that stick

Loop knobs (rounds, compaction, stall timeout, …) live in `~/.kdeps/agent-loop-settings.yaml`. Change via `/model tool set …` or edit the file.

## Workflow next

Fixed HTTP APIs and deterministic pipelines: [Workflow mode](/workflow) (`kdeps run`). Full map: [CLI](/cli) · [Two modes](/modes).
