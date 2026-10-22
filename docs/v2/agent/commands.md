# REPL slash commands

Inside the [agent loop REPL](/agent/), type `/help` for the full list.

Each builtin command is a `commands/<name>.yaml` asset whose `api:` field names the kdeps API operation that runs it. The REPL dispatches through that binding, and the same operation is on the [kdeps API](/reference/api#run-a-slash-command) at `POST /apis/commands.kdeps.io/v1/commands/{name}/call` - see `/apis/commands.kdeps.io/v1/commands` for the list with each operation's display name and usage.

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
| `/model tool set <setting> <value>` | Change a setting, e.g. `set rounds 80` (`0` = unlimited), `set compact-threshold 40k`, `set web-limit 0` (also `bash-limit`/`file-limit`/`code-limit`; `0` removes that per-request call cap), `set retry-delay 5s`, `set stall-timeout 5m`, `set autokill on`. Settings are **persisted** to `~/.kdeps/agent-loop-settings.yaml` and restored next session |
| `/model name [show\|hide\|abbreviate\|auto]` | Show or set how the modeline displays the model name, overriding the theme's default (see [Themes](/agent/repl-themes#themes)); persists |
| `/skills` | List loaded skills |
| `/prompts` | List loaded prompt templates |
| `/<skill-name> [prompt]` | Invoke a skill or prompt template directly |
| `/knowledge [on\|off\|extract]` | Show whether the knowledge vault is on and where it is; `extract` has the local model read every changed note now; or turn per-turn recall, session notes and the knowledge tools on or off (remembered across sessions). See [Knowledge vault](/agent/knowledge) |
| `/recall [words]` | Show the knowledge vault notes recalled for the last turn and why (score, hops, linked from); with words, recall for those instead. See [Knowledge vault](/agent/knowledge) |
| `/compact` | Summarize older turns now, keeping the last few verbatim. Unlike auto-compaction it runs regardless of how large the session is; it only shrinks the saved conversation, not a single turn's own tool output |
| `/history` | Show conversation history |
| `/thinking [off\|minimal\|low\|medium\|high\|xhigh\|auto]` | Enable extended reasoning (Claude only; warns if current model does not support it); persists across sessions |
| `/prompt` | Show the exact LLM request for the last turn (system prompt, messages, tool schemas) |
| `/prompt raw` | Same, unformatted - the raw JSON payload sent to the model |
| `/permission [read-only\|workspace-write\|danger-full-access\|ask]` | Show or set the tool permission mode; persists across sessions (see [Permission modes](/agent/tools#permission-modes)) |
| `/session list\|save\|load\|delete\|checkpoint\|goto\|branches\|import` | Manage this folder's saved sessions and navigate branching history (see [Sessions](/agent/repl-sessions#sessions)) |
| `/editor` | Open current input in `$EDITOR` (ctrl+g) |
| `/copy` | Copy last assistant response to clipboard |
| `/reload` | Reload skills and prompt templates from disk |
| `/theme [name\|list]` | Show or set the REPL's look - `normal` (default), `black`, `linux`, `vim`, `emacs`, or a custom name from `~/.kdeps/themes/` (see [Themes](/agent/repl-themes#themes)); `list` shows built-in and custom names separately; persists. Also `--theme` / `KDEPS_THEME` at startup |
| `/context` | Show current context window size |
| `/context <size>` | Set context window size (e.g. `32768` or `32k`); restarts local model servers with the new `--ctx-size`; persists across sessions |
| `/turo` | Show turo reducer status (state, level). Only available when the `turo` binary is on `PATH` |
| `/turo on\|off` | Enable or disable prompt reduction at runtime |
| `/turo lite\|full\|ultra` | Set the turo compression level |
| `/turo <stage> on\|off` | Toggle a pipeline stage: `filler`, `synonyms`, `gloss`, `defmatch`, `arrows` |
| `/goal` | Show the active goal's task list and status |
| `/goal on\|off` | Enable or disable goal-directed execution entirely; persists across sessions. `off` drops any active goal and runs turns as a plain tool loop |
| `/goal new <text>` | Replace the active goal with a new plan |
| `/goal skip` | Abandon the active task and advance to the next |
| `/goal clear` | Drop the active goal (the next prompt starts a new one) |
| `/auto-improve` | In the kdeps repo, start an instrumented session, run the fixed task list, then read the event log and fix product bugs before the next task |
| `/refine` | Show whether pre-turn prompt refinement is on (see [Prompt refinement](/agent/refine)) |
| `/refine on\|off` | Enable or disable rewriting each prompt for clarity before the turn; on by default, persists across sessions |
| `/efficiency` | Show efficiency enforcement state and limits (see [Efficiency enforcement](/agent/efficiency)) |
| `/efficiency on\|off` | Enable or disable soft-stopping read-only exploration loops; on by default, persists |
| `/efficiency verbose on\|off` | Reveal the hidden model-facing channel for debugging; off by default, persists |
| `/efficiency <setting> <n\|off>` | Set one limit: `reads`, `actions`, `stops`, `tighten`, `web`, `bash`, `file`, `code`; persists |
| `/efficiency preset <name>` | Overwrite every efficiency value from `frugal`, `balanced` or `thorough` |
| `/efficiency reset` | Restore the shipped defaults |
| `/efficiency prompt [auto\|full\|summary]` | After the first full harness send, keep it (`full`), replace each section with its YAML `summary` (`summary`), or do that only on backends that do not cache prompts (`auto`, the default). Persists |
| `/instruct` | Print a kdeps briefing (what kdeps is, how to call a tool with worked `<invoke>` examples, the live tool catalog, memory, goals, runtime feedback) and append it to history as a settled turn, so the next prompt is primed with it |
| `/instruct!` | Same briefing, but sent as a **real turn now** - the model reads it and replies with an acknowledgement on the spot. For models that skim injected context |
| `/instruct <topic>` / `/instruct! <topic>` | Brief on one topic only: `overview`, `modes`, `tools`, `available`, `memory`, `goals`, `feedback`, `files` |
| `/instruct list` | Name the topics without briefing the model |
| `/judges` | Show the judge panel. It is the eval and runs unless turned off (see [Judge panel](/agent/judges)) |
| `/judges add <name> <criteria>` | Add a judge to the explicit roster |
| `/judges remove <name>` | Remove a judge from the explicit roster |
| `/judges grade <name> <kind> <pattern>` | Attach a grader (`exact`, `contains`, `not_contains`, `regex`, `json`, `expr`) to a judge. It runs before that judge's model review |
| `/judges auto [on\|off]` | Show or toggle the per-turn roster; on unless turned off; persists across sessions |
| `/judges off` (or `/judges clear`) | Disable the judge panel (drops the roster, graders included) and store `auto_judges: false` |
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
| `/tools [full\|lean]` | Show or toggle the lean/full tool set (full by default, persists across sessions - see [Lean mode](/agent/tools#lean-mode)) |
| `/upgrade` | Check for a newer kdeps release and, for a standalone install, download/verify/install it (see [Updating kdeps](/agent/repl-updating#updating-kdeps)) |
| `/upgrade nightly` | Same, but checks the nightly channel instead of the latest stable release (see [Nightly builds](/agent/repl-updating#nightly-builds)) |
| `/update [item[@version]...]` | Update harness, tool definitions, themes, recipes, model registries and templates; no args updates everything (see [Updating assets](/agent/repl-updating#updating-assets)) |
| `/update check` \| `/update list` | Show available asset updates \| every asset with its version and state |
| `/update remove <item>...` | Remove assets; restore one with `/update <item>` |
| `/upgrade <version>` | Install an exact version directly, skipping the update check - older than the running build means a downgrade (see [Installing a specific version](/agent/repl-updating#installing-a-specific-version-including-a-downgrade)) |
| `/handshake [on\|off]` | Show or toggle the mandatory session-integrity tool-call check. Turning it on verifies the current model immediately, before your next prompt, then again on every model change/resume/compaction/fold (off by default, persists across sessions - see [Session-integrity handshake](/agent/tools-handshake#session-integrity-handshake)) |
| `/harness [list]` | List harness sections (the system-prompt text sent every turn) with their enabled/disabled state |
| `/harness enable\|disable <name>` | Turn a harness section on/off; persists to `~/.kdeps/harness/<name>.yaml` and takes effect on the next turn (no restart) |
| `/harness events [list]` | List [reactive LLM events](/agent/events) with a trigger summary, action, and enabled/disabled state |
| `/harness events enable\|disable <name>` | Turn an event on/off; persists to `~/.kdeps/events/<name>.yaml` and takes effect immediately - a disabled event's threshold can never fire |
| `/harness preset [list]` | List built-in + user [presets](/agent/events-presets#presets) (e.g. `frugal`, `balanced`, `thorough`) with their descriptions |
| `/harness preset <name>` | Apply a preset: writes its bundled event overrides to `~/.kdeps/events/*.yaml` and reloads immediately - no restart |
| `/harness reminders [list]` | List every harness section with its reminder on/off state (see [Harness reminders](/agent/tools-harness#harness-reminders)) |
| `/harness reminders <name> on\|off` | Force that section's text onto every LLM prompt and every tool call result, independent of its own enabled/disabled state or normal trigger; persists to `~/.kdeps/harness/<name>.yaml`. A `reminder:<names>` segment appears in the status line (next to `turo:`) while any are active, and disappears when the last one is turned off |
| `/harness reminders instruct:<topic> on\|off` | Same, for an `/instruct` topic (`overview`, `modes`, `tools`, `available`, `memory`, `goals`, `feedback`, `files`); a bare topic name also works unless a harness section shares it (`tools`, `memory`). Persists to `~/.kdeps/agent-loop-settings.yaml` |
| `/konfig export\|import [path]` | Export or import tuning, harness, themes, events, actions, presets, and skills as a self-contained YAML file (default `./konfig.yaml`) - see [konfig](/agent/konfig) |
| `/login` | m365 backend only: open a browser window to (re-)sign in, even if a session is already cached (see [M365 Copilot](/llm-server/m365)) |

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

## Text files and read-only commands

The same scan finds existing, readable text files by name (`look at "main.go"`) and offers to inline them like `@main.go` would - images/binaries are never auto-detected, use an explicit `@path` instead. Only a strict allowlist of read-only commands is offered (`ls`, `df`, `ps`, `git status`, `go env`, `docker ps`, etc.); destructive commands (`rm`, `git commit`, `go build`, `docker rm`, ...) never match, even quoted. One confirmation covers everything in a message; declining (or pressing Enter) sends your text unchanged.

## Pipes and command substitution

Pipes and command substitution are recognized too, as long as every stage is allowlisted: `"ps aux | grep -i kdeps"` runs as one pipeline provided each `|`-separated stage is read-only (a stage like `xargs rm` invalidates the whole thing); `` $(git rev-parse HEAD) `` needs no extra quoting - its `$(...)` body is checked the same way, and anything that could chain a second command inside the parens (`;`, `&`, `` ` ``, a nested `$(`) is rejected outright.

## Auto-context on and off

Disable it for the session with `/autocontext off` if the confirmation prompt gets in your way; `/autocontext on` re-enables it, and `/autocontext` alone shows the current state.

## Startup flags

| Flag | Effect |
|------|--------|
| `--resume <id>` | Resume a specific session directly, skipping the picker |
| `--new` | Start a clean session, skipping the resume picker for this folder |
| `--index <folder>` | Search a folder outside the project with it: added to the card and indexed before the session starts (repeatable) |
| `--model` / `--backend` / `--base-url` | Override the model for this run |
| `--system <text>` | Prepend a system prompt |
| `--theme <name>` | REPL theme: `normal` (default), `black`, `linux`, `vim`, `emacs`, or a custom name from `~/.kdeps/themes/` (see [Themes](/agent/repl-themes#themes)) |

Sessions and memory are stored under `~/.kdeps/`, partitioned by the directory
`kdeps` runs in. See [Sessions](/agent/repl-sessions#sessions).

## Auto-improve

`/auto-improve` runs a fixed task list against this repo and fixes product bugs the instrument log shows.

```d2
direction: down
start: "/auto-improve" {shape: oval}
repo: "module github.com/kdeps/kdeps/v2?" {shape: diamond}
session: "record ~/.kdeps/instrument/auto-improve"
task: "model runs the next task"
eval: "model reads events.ndjson\nand fixes product bugs"
more: "another task?" {shape: diamond}
done: "done" {shape: oval}
start -> repo
repo -> session: "yes"
repo -> done: "other folder"
session -> task
task -> eval
eval -> more
more -> task: "yes"
more -> done: "no"
```

The working directory must be this module, or a folder inside it. Another module stops the walk. When no session is recording, the command starts one named `auto-improve`. An existing session keeps recording. Each task asks for one tool use and does not edit files. The evaluate turn reads `session.md` and `events.ndjson` (`kind` on each line). It fixes a loop, a refusal of work the tools can do, a call to the wrong tool, or a wrong tool error, and adds a test. A model guess is left as a guess. The session stays open after the command finishes.

The three tasks are: read `README.md` and name the two modes, run `git status --short`, and find the `instrument.Start` signature.

## Declared commands

Every slash command is a `commands` [asset](/agent/repl-updating#updating-assets): one YAML file that `kdeps update` keeps current. For a command kdeps runs itself, the file supplies the `/help` text, aliases and tab completion and names the operation that runs it in `api:`:

```yaml
# commands/model.yaml - a builtin command's file
version: 1.0.1
name: model
order: 20
builtin: true            # kdeps runs it; this file supplies help, aliases and completion
api: model               # the kdeps API operation that runs it: the file name
usage:
  - args: "[name]"
    help: Show or switch LLM model mid-session
```

Any other file *is* the command: kdeps runs its `run:` steps in order, so a new command ships as an asset update, with no new binary.

```d2
direction: right
input: "/review error handling" {shape: oval}
lookup: "commands/review.yaml"
step1: "step: command\n(another slash command)"
step2: "step: prompt\n(sent to the model as a turn)"
reply: "model reply" {shape: oval}
input -> lookup: "name or alias"
lookup -> step1: "run: in order"
step1 -> step2
step2 -> reply
```

```yaml
# commands/review.yaml - shipped with kdeps; update with `kdeps update commands/review`
version: 1.0.0            # bump on every change; kdeps update installs newer versions
name: review              # typed as /review
order: 520                # position in /help (lower first)
aliases: [rv]             # optional: /rv runs it too
usage:                    # one /help line each; the first is the command palette summary
  - args: "[focus]"
    help: Review the uncommitted changes in this repository
run:                      # steps run in order; the first failing step stops the rest
  - command: /efficiency preset thorough   # any slash command, including other declared ones
  - prompt: |-                             # sent to the model as a user turn
      Review the uncommitted changes. Extra focus, if given: $ARGUMENTS
```

- `$1`, `$2`, ... and `$ARGUMENTS` in a step are replaced by the command's arguments, the same placeholders [prompt templates](/agent/skills) use (`${1:-default}` gives a fallback).
- A declared command cannot take a built-in's name or alias, and commands calling commands stop after 8 levels.
- A builtin file's `api:` must be its own file name, and that name must be one the binary implements. A mismatch is skipped with a warning instead of running another command's handler.
- Built-in command files cannot be removed. Declared ones can, with `kdeps uninstall commands/<name>`, and `kdeps update commands/<name>` restores them.

| Shipped declared command | Runs |
|---|---|
| `/frugal` | `/efficiency preset frugal` |
| `/thorough` | `/efficiency preset thorough` |
| `/review [focus]` | A prompt asking the model to review the uncommitted changes, without editing files |
| `/explain <target>` | A prompt asking the model to read the code and explain how `<target>` works |

## See also

- [Agent mode](/agent/) - overview and starting the REPL
- [Built-in tools](/agent/tools) - tools available to the model
- [Goal-directed execution](/agent/goals)

## Declared commands on the API

A declared command (a `commands/` asset with `run:`) is also served on the commands API under its own name. The call runs the same steps as `/name` in the REPL, against the active session, and returns each model reply in order. `command:` steps run in the REPL and print their output to the terminal rather than returning it.
