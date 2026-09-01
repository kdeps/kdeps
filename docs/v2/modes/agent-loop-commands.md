# REPL slash commands

Inside the [agent loop REPL](/modes/agent-loop-mode), type `/help` for the full list:

| Command | Description |
|---------|-------------|
| `/help` | Show available commands |
| `/clear` | Summarize and clear the current conversation |
| `/model [name]` | Show or switch LLM model mid-session (tab-complete shows up to 10 suggestions) |
| `/model default [name]` | Show or set the default startup model, persisted to `~/.kdeps/agent-loop-settings.yaml` |
| `/model list` | List all available models with provider status |
| `/model ps` | List running local model servers (llamafile/gguf) with PID, port, and health |
| `/model ps kill <model>` | Kill a running local model server and clean up its port file |
| `/model ps switch <model>` | Switch the active model to a running local server |
| `/model hff search <query>` | Search HuggingFace for GGUF repos (sorted by downloads) |
| `/model hff info <repo>` | List GGUF files and sizes available in a HuggingFace repo |
| `/model hff download <repo> [file]` | Download a GGUF from HuggingFace; auto-registers an alias for `/model` |
| `/model tool [list]` | Show agent loop settings: tool rounds, retries, retry delay, compaction, history caps, stall timeout, auto-allocation |
| `/model tool set <setting> <value>` | Change a setting, e.g. `set rounds 80` (`0` = unlimited), `set compact-threshold 40k`, `set retry-delay 5s`, `set stall-timeout 5m`, `set autokill on`. Settings are **persisted** to `~/.kdeps/agent-loop-settings.yaml` and restored next session |
| `/skills` | List loaded skills |
| `/prompts` | List loaded prompt templates |
| `/<skill-name> [prompt]` | Invoke a skill or prompt template directly |
| `/compact` | Summarize history to free context |
| `/history` | Show conversation history |
| `/thinking [off\|minimal\|low\|medium\|high\|xhigh\|auto]` | Enable extended reasoning (Claude only; warns if current model does not support it); persists across sessions |
| `/prompt` | Show the exact LLM request for the last turn (system prompt, messages, tool schemas) |
| `/prompt raw` | Same, unformatted - the raw JSON payload sent to the model |
| `/permission [read-only\|workspace-write\|danger-full-access\|ask]` | Show or set the tool permission mode; persists across sessions (see [Permission modes](/modes/agent-loop-tools#permission-modes)) |
| `/session list\|save\|load\|delete\|checkpoint\|goto\|branches\|import` | Manage saved sessions and navigate branching history |
| `/editor` | Open current input in `$EDITOR` (ctrl+g) |
| `/copy` | Copy last assistant response to clipboard |
| `/reload` | Reload skills and prompt templates from disk |
| `/stealth [on\|off]` | Muted UI - render everything in dark gray with the model name barely visible (for use in public); persists to `~/.kdeps/agent-loop-settings.yaml`. Also `--stealth` / `KDEPS_STEALTH=1` at startup |
| `/context` | Show current context window size |
| `/context <size>` | Set context window size (e.g. `32768` or `32k`); restarts local model servers with the new `--ctx-size`; persists across sessions |
| `/turo` | Show turo reducer status (state, level). Only available when the `turo` binary is on `PATH` |
| `/turo on\|off` | Enable or disable prompt reduction at runtime |
| `/turo lite\|full\|ultra` | Set the turo compression level |
| `/turo <stage> on\|off` | Toggle a pipeline stage: `filler`, `synonyms`, `gloss`, `defmatch`, `arrows` |
| `/goal` | Show the active goal's task list and status |
| `/goal new <text>` | Replace the active goal with a new plan |
| `/goal skip` | Abandon the active task and advance to the next |
| `/goal clear` | Drop the active goal |
| `/judges` | Show the configured judge panel (reviews each turn's final output - see [Judge panel](/modes/agent-loop-judges)) |
| `/judges add <name> <criteria>` | Add a judge to the explicit roster |
| `/judges remove <name>` | Remove a judge from the explicit roster |
| `/judges auto [on\|off]` | Show or toggle a per-turn auto-generated roster; persists across sessions |
| `/judges clear` | Disable the judge panel (drops the explicit roster and turns off auto-judges) |
| `/memory` | Show memory store overview: entry count and the 10 most recently updated entries (with values) |
| `/memory list` | List every stored memory entry with a truncated value preview |
| `/memory search <query>` | Search memory keys and values for a substring |
| `/memory show <key>` | Show one entry's full value, type, timestamps, and its dependency graph node (the same `<graph-node>` block the model receives in its prompt) |
| `/settings` | Open the tool/skill selector |
| `/exit` | Exit the REPL |
| `! <cmd>` | Run a shell command; the output becomes an agent turn - the model responds and can act on it (e.g. `!make lint` -> the model fixes the findings) |
| `!! <cmd>` | Run a shell command silently - no LLM turn, nothing added to context |
| `@<path>` | Inline a file's contents (text) or attach it (image) into the next turn, e.g. `explain @main.go` |
| `/autocontext [on\|off]` | Show or toggle auto-detecting command/file mentions in plain chat text (on by default, persists across sessions) |
| `/tools [full\|lean]` | Show or toggle the lean/full tool set (full by default, persists across sessions - see [Lean mode](/modes/agent-loop-tools#lean-mode)) |
| `/upgrade` | Check for a newer kdeps release and, for a standalone install, download/verify/install it (see [Updating kdeps](/modes/agent-loop-mode#updating-kdeps)) |
| `/upgrade nightly` | Same, but checks the nightly channel instead of the latest stable release (see [Nightly builds](/modes/agent-loop-mode#nightly-builds)) |
| `/login` | m365 backend only: open a browser window to (re-)sign in, even if a session is already cached (see [M365 Copilot](/reference/llm-providers-m365)) |

## Auto-detected commands and files

`!cmd` and `@path` require you to know kdeps' own syntax. Auto-context detection covers the common case where you just describe what you want in plain English - but only fires on a command or file mention wrapped in quotes (single or double); an unquoted mention in plain prose is never offered:

```text
you type: "can you run \"df -h\" and take a look at \"main.go\"?"
              |
              v
kdeps scans the quoted spans for a read-only command ("df -h")
and an existing, readable file ("main.go")
              |
              v
"Detected in your message:
   command: df -h
   file: main.go
 Run the command(s) / include the file(s)? [y/N]"
              |
       y -----+----- n/Enter
       |             |
       v             v
runs `df -h`,   sends your original
inlines         text completely
main.go, sends  unchanged
the turn
```

Only text inside `"..."` or `'...'` is ever scanned - typing `can you check df -h for me` with no quotes triggers nothing, even though `df -h` is a recognized command. Quoting `"df -h"` is what tells auto-context you mean it literally.

The same scan finds existing, readable **text files** by name (`look at "main.go"`) and offers to inline them like `@main.go` would - images/binaries are never auto-detected, use an explicit `@path` instead. Only a strict allowlist of read-only commands is offered (`ls`, `df`, `ps`, `git status`, `go env`, `docker ps`, etc.); destructive commands (`rm`, `git commit`, `go build`, `docker rm`, ...) never match, even quoted. One confirmation covers everything in a message; declining (or pressing Enter) sends your text unchanged.

**Pipes and command substitution** are recognized too, as long as every stage is allowlisted: `"ps aux | grep -i kdeps"` runs as one pipeline provided each `|`-separated stage is read-only (a stage like `xargs rm` invalidates the whole thing); `` $(git rev-parse HEAD) `` needs no extra quoting - its `$(...)` body is checked the same way, and anything that could chain a second command inside the parens (`;`, `&`, `` ` ``, a nested `$(`) is rejected outright.

Disable it for the session with `/autocontext off` if the confirmation prompt gets in your way; `/autocontext on` re-enables it, and `/autocontext` alone shows the current state.

## See also

- [Agent loop mode](/modes/agent-loop-mode) - overview and starting the REPL
- [Built-in tools](/modes/agent-loop-tools) - tools available to the model
- [Goal-directed execution](/modes/agent-loop-goals)
