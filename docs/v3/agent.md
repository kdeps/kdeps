# Agent mode

An interactive LLM loop. The model plans tasks, calls tools, and drives work to completion.

## Start

```bash
kdeps                              # model-only REPL
kdeps ./my-agent/                  # one workflow = one tool
kdeps ./agents/                    # every workflow in the folder = a tool
kdeps --model deepseek-v4-flash --system "You are a DevOps assistant."
kdeps --resume <session-id>        # continue a saved session
```

When you pass a path, workflows (and agencies) under it are registered as tools. If the model calls one, the **full workflow engine** runs — same DAG as `kdeps run`.

## Built-in tools

Always available in the loop (examples): web search, scraper, calculator, SQL helpers, bash, file ops. New loop-only capabilities land here; user workflows stay as tools, not as a second executor path.

## Goal-directed runs

A prompt becomes a task list. The model works the active task until it finishes or fails, then moves on.

```text
prompt -> tasks -> [task 1] -> [task 2] -> ... -> answer
```

Tools like `task_complete` / `task_fail` advance the plan. Slash command `/goal` inspects or steers it.

## Useful slash commands

| Command | What it does |
|---------|----------------|
| `/help` | Full command list |
| `/model [name]` | Show or switch model |
| `/model list` | Available models |
| `/model ps` | Running local model servers |
| `/clear` | Summarize and clear conversation |
| `/compact` | Compact history |
| `/session …` | Save / load / list sessions |
| `/thinking …` | Extended reasoning (when supported) |
| `/turo …` | Token reducer (if `turo` is on `PATH`) |
| `/goal …` | View or change the task plan |
| `/exit` | Leave the REPL |
| `! <cmd>` | Run a shell command; model sees the output |
| `!! <cmd>` | Run a shell command; no model turn |

Type `/help` for the full set — it stays in sync with the binary.

## Skills and prompts

```bash
kdeps --skill ~/.kdeps/skills/
```

Skills teach the agent how to work in a domain (including how to scaffold kdeps projects). List loaded skills with `/skills`.

## Settings that stick

Loop settings (tool rounds, compaction, stall timeout, …) persist under `~/.kdeps/agent-loop-settings.yaml`. Change them with `/model tool set …` or edit the file.

Next: [Two modes](/modes) · [Workflow mode](/workflow) · [CLI](/cli).
