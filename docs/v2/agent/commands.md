# REPL slash commands

Inside the [agent loop REPL](/agent/), type `/help` for the full list.

Each builtin command is a `commands/<name>.yaml` asset whose `api:` field names the kdeps API operation that runs it. The REPL dispatches through that binding, and the same operation is on the [kdeps API](/reference/api#run-a-slash-command) at `POST /apis/commands.kdeps.io/v1/commands/{name}/call` - see `/apis/commands.kdeps.io/v1/commands` for the list with each operation's display name and usage.

| Command | Description |
|---------|-------------|
| `/help` | Show available commands |
| `/clear` | Summarize and clear the current conversation |
| `/model [name]` | Show or switch LLM model mid-session (tab-complete shows up to 10 suggestions) |
| `/model record [clear]` | Show the current model's record of caught refusals (sent to it each turn), or clear it |
| `/model default [name]` | Show or set the default startup model, persisted to `~/.kdeps/agent-loop-settings.yaml` |
| `/model list` | List all available models with provider status |
| `/model ps` | List running local model servers (llamafile/gguf) with PID, port, and health |
| `/model ps kill <model>` | Kill a running local model server and clean up its port file |
| `/model ps switch <model>` | Switch the active model to a running local server |
| `/model hff search <query>` | Search HuggingFace for GGUF repos (sorted by downloads) |
| `/model hff info <repo>` | List GGUF files and sizes available in a HuggingFace repo |
| `/model hff download <repo> <file>` | Download a GGUF from HuggingFace; auto-registers an alias for `/model` |
| `/model tool [list]` | Show agent loop settings: tool rounds, retries, retry delay, compaction, history caps, stall timeout, auto-allocation |
| `/model tool set <setting> <value>` | Change a setting, e.g. `set rounds 80` (`0` = unlimited), `set compact-threshold 40k`, `set web-limit 0` (also `bash-limit`/`file-limit`/`code-limit`; `0` removes that per-request call cap), `set retry-delay 5s`, `set stall-timeout 5m`. Settings are **persisted** to `~/.kdeps/agent-loop-settings.yaml` and restored next session |
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
| `/auto-improve` | In the kdeps repo, record a session, jump across OpenRouter free models, fix product bugs the event log shows, and file one GitHub issue |
| `/refine` | Show whether pre-turn prompt refinement is on (see [Prompt refinement](/agent/refine)) |
| `/refine on\|off` | Enable or disable rewriting each prompt for clarity before the turn; off by default, persists across sessions. A session that already stored refine-on stays on |
| `/efficiency` | Show efficiency enforcement state and limits (see [Efficiency enforcement](/agent/efficiency)) |
| `/efficiency on\|off` | Enable or disable soft-stopping read-only exploration loops; on by default, persists |
| `/efficiency verbose on\|off` | Reveal the hidden model-facing channel for debugging; off by default, persists |
| `/efficiency <setting> <n\|off>` | Set one limit: `reads`, `actions`, `stops`, `tighten`, `web`, `bash`, `file`, `code`; persists |
| `/efficiency preset <name>` | Overwrite every efficiency value from `frugal`, `balanced` or `thorough` |
| `/efficiency reset` | Restore the shipped defaults |
| `/efficiency prompt [auto\|full\|summary]` | After the first full harness send, keep it (`full`), replace each section with its YAML `summary` (`summary`), or do that only on backends that do not cache prompts (`auto`, the default). Persists |
| `/instruct` | Print a kdeps briefing (what kdeps is, how to call a tool with worked `<invoke>` examples, the live tool catalog, memory, runtime feedback) and append it to history as a settled turn, so the next prompt is primed with it |
| `/instruct!` | Same briefing, but sent as a **real turn now** - the model reads it and replies with an acknowledgement on the spot. For models that skim injected context |
| `/instruct <topic>` / `/instruct! <topic>` | Brief on one topic only: `overview`, `modes`, `tools`, `available`, `memory`, `feedback`, `files` |
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
| `/harness reminders instruct:<topic> on\|off` | Same, for an `/instruct` topic (`overview`, `modes`, `tools`, `available`, `memory`, `feedback`, `files`); a bare topic name also works unless a harness section shares it (`tools`, `memory`). Persists to `~/.kdeps/agent-loop-settings.yaml` |
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
jump: "OpenRouter: another free model"
task: "model runs the next task"
eval: "model reads events.ndjson\nand fixes product bugs"
more: "another task?" {shape: diamond}
issue: "one GitHub issue"
done: "done" {shape: oval}
start -> repo
repo -> session: "yes"
repo -> done: "other folder"
session -> jump
jump -> task
task -> eval
eval -> more
more -> jump: "yes"
more -> issue: "no"
issue -> done
```

The working directory must be this module, or a folder inside it. Another module stops the walk. When no session is recording, the command starts one named `auto-improve`. An existing session keeps recording. On an OpenRouter backend, or when the current model is an OpenRouter free id, each task and each evaluate turn starts by printing `auto-improve: model <id>` and switching to another free id at random. A local backend stays on the model it was started with. A provider 429 during the capture continues on a random free id. So does a free id the provider refuses (a 400 or 502 "Provider returned error", a 404 "No endpoints found" for tool use, or a 403 "only available on agentic harnesses"). Each task asks for one tool use and does not edit files. The evaluate turn reads `session.md` and `events.ndjson` (`kind` on each line). It fixes a loop, a refusal of work the tools can do, a call to the wrong tool, or a wrong tool error, and adds a test. A model guess is left as a guess. When the command returns, it files one GitHub issue and prints `auto-improve: issue <url>`. If every failure in the session was a provider refusal or rate limit, no issue is filed and it prints `auto-improve: no issue (provider-only failures)`. A kdeps error, a tool error, or a clean run still files one. The issue lists up to 40 events that have an error field. The session stays open after the command finishes.

The three tasks are: read `README.md` and name the two modes, run `git status --short`, and find the `instrument.Start` signature.

`/auto-improve 2` runs only task 2, then its evaluate, in the current session. The number is 1, 2, or 3.

A daily workflow (`.github/workflows/bug-hunt.yml`, 07:30 UTC) runs one task per process in the kdeps repository. It passes `--backend openrouter`, `--instrument=bug-hunt-N`, and a model id from `assets/models/openrouter.yaml`. `scripts/bug-hunt.sh run` shuffles that list. `scripts/bug-hunt.sh models` stays in catalog order. Each process is hunt N of 3: the task, then the evaluate. The instrument log holds only that hunt. Inside a hunt, free-model requests wait so they stay under the account per-minute cap (`openrouter-free-per-minute`, 20 calls, every free id together). An account 429 waits out the minute and retries the same id. When the process does not print `auto-improve: done`, it tries another id, up to four. If the log has no `auto-improve: issue http` line and no `auto-improve: no issue` line, the job files the issue. A tree with edits is pushed as its own pull request with auto-merge, which merges after the required checks pass. The job then checks out the branch it started on and runs the next hunt. The workflow needs the `OPENROUTER_API_KEY` and `RELEASE_TOKEN` repository secrets.

```bash
id=$(grep -m1 'id: "' assets/models/openrouter.yaml | sed -E 's/.*"([^"]+)".*/\1/')
printf '/auto-improve 1\n/exit\n' | kdeps --backend openrouter --model "$id" --instrument=bug-hunt-1
```

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
step1: "step: command or tool\n(a slash command or a kdeps API tool)"
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

A step can also call a kdeps API tool directly. The tool's output is the step's output; the arguments go in `args:`, and their string values take the same placeholders.

```yaml
# ~/.kdeps/commands/calc.yaml - /calc 6*7 prints 42
version: 1.0.0
name: calc
usage:
  - args: "<expression>"
    help: Evaluate an expression with the calculator tool
run:
  - tool: calculator          # any tool on /apis/tools.kdeps.io, shipped or your own
    args:
      expression: "$1"        # the first argument of /calc
```

<div v-pre>

A tool step's `print:` is a Go template over the tool's result: the fields of a JSON object, or `.output` for any other text. It can call `join` and `trunc` (`{{trunc 80 .content}}` cuts to 80 characters plus `...`). `/clear`, `/compact` and `/history` are built this way, on the `session_ops` tool, which reads and changes the live session:

```yaml
# commands/history.yaml (shipped)
version: 2.0.0
requires:
  tools/session_ops: ""     # kdeps update installs the tool with the command
name: history
usage:
  - args: ""
    help: Show recent conversation turns
run:
  - tool: session_ops       # served on /apis/tools.kdeps.io; not offered to the model
    args:
      operation: messages   # returns {"turns": 1, "messages": [{"turn": 0, "role": "user", "content": "..."}]}
    print: |-
      {{if not .messages}}No conversation history.{{else}}**Conversation history ({{.turns}} turns):**
      {{range .messages}}
      - [{{.turn}}] {{if eq .role "user"}}YOU{{else}}AGENT{{end}}: {{trunc 80 .content}}{{end}}{{end}}
```

| `session_ops` operation | Arguments | Result |
|---|---|---|
| `messages` | - | `turns`, `messages` (`turn`, `role`, `content`) |
| `clear` | `summarize_min_turns`: summarize the branch first when the session has at least this many turns (`0` never) | `turns` (before clearing), `summary` |
| `compact` | - | `summary` (empty when there were too few turns), `turns`, `keep` (recent turns kept verbatim) |
| `last_reply` | - | `text` (the latest assistant message, empty when there is none) |
| `get` | `key`: a session setting (see [Setting commands](#setting-commands)) | the setting's state: `value` plus its extra fields |
| `set` | `key`, `value`: one of the values the setting's command file lists | the new state; persisted across sessions |
| `sessions` | - | `sessions` (`id`, `name`, `prompt`, `model`, `turns`, `updated`): this folder's saved sessions |
| `save` | `name` (optional) | `id`, `name` |
| `load` | `id` | `id`, `turns`; the session's model is restored and the handshake required |
| `delete` | `id` | `id` |
| `import` | `path` (a JSONL file; `~/` is the home folder) | `id`, `turns`; the imported session is loaded |
| `checkpoint` | - | `id` (empty when the session has no messages) |
| `goto` | `id`: an entry id | `id`, `found`, `turns` |
| `branches` | - | `branches` (`branch`, `at`, `ids`) |
| `skills` | - | `skills` (`name`, `description`): behind `/skills` |
| `prompts` | - | `prompts` (`name`, `hint`, `description`): behind `/prompts` |
| `memory_entries` | `query` (search; given but empty is an error), `order` (`recent`), `limit` | `total`, `more` (left out by `limit`), `query`, `entries` (`key`, `type`, `value`): behind `/memory`, `/memory list`, `/memory search` |
| `memory_entry` | `key` | `key`, `type`, `value`, `related`, `updated`, `graph` (the `<graph-node>` block the model sees): behind `/memory show` |
| `judges` | - | `judges` (`name`, `criteria`, `graders`: `kind`, `pattern`), `auto`, `last` (the last generated roster): behind `/judges` |
| `judge_add` | `name`, `criteria` | `name` |
| `judge_remove` | `name` | `name`, `removed` |
| `judge_grade` | `name`, `kind` (`exact`, `contains`, `not_contains`, `regex`, `json`, `expr`), `pattern` | `name`, `kind` |
| `judges_clear` | - | `cleared`: no roster and no auto roster, saved |
| `harness_sections` | - | `sections` (`name`, `kind`, `order`, `max`, `preamble`, `enabled`): behind `/harness` |
| `harness_set` | `name`, `enabled` (`on` or `off`) | `name`, `enabled`; saved to `~/.kdeps/harness/<name>.yaml` |
| `harness_events` | - | `events` (`name`, `on`: the trigger fields that are set, `items`, `run`, `enabled`): behind `/harness events` |
| `event_set` | `name`, `enabled` | `name`, `enabled`; saved to `~/.kdeps/events/<name>.yaml` |
| `presets` | - | `presets` (`name`, `description`) |
| `preset_apply` | `name` | `name`, `harness`, `events` (override counts) |
| `harness_reminders` | - | `reminders` (`name`, `on`), harness sections and `instruct:<topic>` |
| `reminder_set` | `name`, `enabled` | `name`, `enabled` |
| `efficiency` | - | `value` (on/off), `verbose`, `prompt`, `limits` (`reads`, `actions`, `stops`, `failures`, `tighten`, `web`, `bash`, `file`, `code`; `off` when disabled): behind `/efficiency` |
| `efficiency_limit` | `setting`, `value` (a count or `off`) | `setting`, `value`; saved to that limit's event |
| `efficiency_reset` | - | `reset`: every efficiency event back to its shipped default |
| `recall` | `query` (optional: the last turn's recall without it) | `seeds`, `hits` (`path`, `score`, `hops`, `via`, `summary`), `tokens`: behind `/recall` |
| `knowledge` | - | `value` (on/off), `vault`, `graph`: behind `/knowledge` |
| `knowledge_extract` | - | `output`: the local model reads the changed notes and the graph is rebuilt |
| `last_request` | `raw` (`true` for the request as JSON) | `messages` (`index`, `role`, `tokens`, `content`), `tools` (`name`, `tokens`, `description`), `tool_tokens`, `tokens`; or `raw`: behind `/prompt` |
| `search_index` | - | `output`: the working project's folder indexes brought up to date |
| `search_folders` | - | `project`, `places`, `folders` (`path`, `indexed`) |
| `search_folder` | `action` (`add` or `remove`), `folder` | `output` |
| `commands_help` | - | `commands` (`usage`, `help`): every installed command's usage lines, behind `/help` |
| `konfig_export` | `path` | `path` and the `harness`, `themes`, `events`, `actions`, `skills` counts written |
| `konfig_import` | `path` | the same counts, read and applied |
| `pipelines` | - | `/kartographer`'s live numbers: `sent`, `generated`, `total`, `web`/`bash`/`file`/`code` (`calls`, `limit`, `reached`), `turns`, `threshold`, `memory` |
| `reload` | - | `skills`, `prompts` (counts): rereads skills and prompt templates from disk; behind `/reload` |
| `model` | `name`: a model, a `.gguf`/`.llamafile` URL, or an OpenAI-compatible base URL | `model`, and `registered`, `ambiguous` + `typed`, or `closest` + `typed` when the name resolved that way |
| `model_pick` | `filter` (optional) | `model`, `picked`: opens the model picker when there is one |
| `model_default` | `name` (optional) | `model`; with no name, `configured` |
| `models` | - | `current`, `cloud` (`backend`, `env`, `ready`, `models`: `id`, `desc`, `current`), `local` (`id`, `downloaded`, `repo`, `current`) |
| `model_favorite` | `name`, `on` (`on` or `off`) | `name`, `on`, `warning` when it could not be saved |
| `loop_settings` | - | one field per `/model tool` setting |
| `loop_set` | `setting`, `value` | `setting`, `value` as shown (`unlimited`, `disabled`, ...) |
| `servers` | - | `servers` (`pid`, `port`, `backend`, `model`, `status`) |
| `server_kill` | `model` | `model` |
| `hf_download` | `repo`, `file` | `path`, `alias` |
| `tool` | `tool`: a tool that lives only in the agent session (`judge_verdict`, `knowledge_recall`, `knowledge_card_get`, `knowledge_card_update`, `knowledge_note_save`, `load_skill`, `identity_get`, `session_handshake`); `args`: its arguments | that tool's answer. A tool with its own route (`bash_exec`, `read_file`, ...) is refused |

A tool step's `as:` names its result for the steps after it. Their string `args` and `print:` templates read it as `{{.steps.<as>.<field>}}`. `/copy` hands the last reply to the clipboard this way:

```yaml
# commands/copy.yaml (shipped)
run:
  - tool: session_ops
    args:
      operation: last_reply
    as: reply                         # saves {"text": "..."} as .steps.reply
  - tool: clipboard_copy              # a python: tool asset; the model can call it too
    args:
      text: "{{.steps.reply.text}}"   # an args string with {{ is rendered over the earlier steps
    print: >-
      {{if .copied}}Copied last response to clipboard.{{else}}Nothing to copy: no assistant response in session.{{end}}
```

</div>

- `$1`, `$2`, ... and `$ARGUMENTS` in a step are replaced by the command's arguments, the same placeholders [prompt templates](/agent/skills) use (`${1:-default}` gives a fallback).
- A declared command cannot take a built-in's name or alias, and commands calling commands stop after 8 levels.
- A builtin file's `api:` must be its own file name, and that name must be one the binary implements. A mismatch is skipped with a warning instead of running another command's handler.
- Built-in command files cannot be removed. Declared ones can, with `kdeps uninstall commands/<name>`, and `kdeps update commands/<name>` restores them.

A command you write yourself goes in `~/.kdeps/commands/`. kdeps loads it at start (and after `kdeps update`), with no registry or lock entry, and serves it on the commands API like a shipped one. A file there with a shipped command's name replaces it.

```yaml
# ~/.kdeps/commands/standup.yaml - typed as /standup
version: 1.0.0
name: standup
order: 900
usage:
  - args: "[since]"
    help: Summarize my git activity for a standup
run:
  - prompt: |-
      Run git log for my commits since ${1:-yesterday} and summarize them in three bullets.
```

| Shipped declared command | Runs |
|---|---|
| `/frugal` | `/efficiency preset frugal` |
| `/thorough` | `/efficiency preset thorough` |
| `/review [focus]` | A prompt asking the model to review the uncommitted changes, without editing files |
| `/explain <target>` | A prompt asking the model to read the code and explain how `<target>` works |

## Setting commands

A command file with a `setting:` block (instead of `run:`) shows or sets one session setting. kdeps owns the setting itself (the live REPL state and saving it across sessions); the accepted values and every line the command prints are in the file. The block reads and writes through the `session_ops` tool's `get` and `set` operations, so any kdeps API caller can do the same, and `set` accepts only the values the command file lists.

```text
/refine        -> prints show:     (current value)
/refine off    -> stores "off", saves it, prints set:
/refine maybe  -> changes nothing, prints invalid:
```

<div v-pre>

```yaml
# commands/refine.yaml (shipped, wording abridged)
version: 2.0.0
name: refine
usage:
  - args: "[on|off]"
    help: Show or toggle pre-turn prompt refinement (off by default)
setting:
  key: refine        # the session setting: refine, handshake, autocontext, permission, tools, thinking, theme, turo, fold, fold-threshold, fold-items, model-name, auto-judges, efficiency, efficiency-verbose, efficiency-prompt, knowledge or context
  values: [on, off]  # accepted arguments, matched without case
  show: >-           # Go template; .value is the current value
    {{if eq .value "on"}}prompt refinement is on{{else}}prompt refinement is off - /refine on to enable{{end}}
  set: "prompt refinement {{.value}}"     # after storing; .value is the new value
  invalid: "Usage: /refine [on|off]"     # .arg is what was typed; .values the accepted values
```

</div>

| Shipped setting command | Setting | Values | Extra template fields |
|---|---|---|---|
| `/refine` | `refine` | `on`, `off` | - |
| `/handshake` | `handshake` | `on`, `off` | - |
| `/autocontext` | `autocontext` | `on`, `off` | - |
| `/permission` (alias `/permissions`) | `permission` | `read-only`, `workspace-write`, `danger-full-access`, `ask` | - |
| `/tools` | `tools` | `full`, `lean` | `.count` (tools in the set), `.available` (false when lean mode trimmed nothing) |
| `/thinking` | `thinking` | `off`, `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, `auto` | `.budget` (token budget), `.return`, `.model`, `.supported` (false when the model may not reason) |
| `/theme` | `theme` | every built-in and `~/.kdeps/themes` name (no `values:` needed) | `.builtin`, `.custom` (name lists) |
| `/turo` | `turo` | `on`, `off`, `lite`, `full`, `ultra`, `wenyan`, `<stage> on`/`<stage> off` | `.available` (false when turo is not installed), `.level`, one field per stage (`.filler`, `.synonyms`, `.gloss`, `.defmatch`, `.arrows`) |

Two more fields shape what the command accepts:

```yaml
setting:
  key: turo
  values: [on, off, "gloss on", "gloss off"]  # every argument typed is joined by one space: /turo gloss off
  showOn: [list]                              # these arguments show the setting like a bare /turo
```

<div v-pre>

Templates can call `join` (`{{join .builtin ", "}}`).

</div>

Copy one to `~/.kdeps/commands/` to change its wording or narrow its values. A `key` kdeps does not have is skipped with a warning at load.

A numeric setting takes a value of its form instead of a list: `fold-threshold` takes a token count (`3000`, `3k`) and `fold-items` a positive integer, both or `default`, which clears the override so the `fold` event's value applies. Their blocks need no `values:`.

## Subcommands

A command file with `subcommands:` (instead of `run:` or `setting:`) dispatches on its first argument. Each entry is itself a `run:`, a `setting:` or more `subcommands:`, and gets the remaining arguments. The `""` entry runs when no argument is given; an unknown word is an error that names the choices.

```text
/fold                -> subcommands."" (status)
/fold threshold 3k   -> subcommands.threshold: setting fold-threshold, value "3k"
/fold preset tight   -> subcommands.preset.subcommands.tight: two run steps
/fold preset huge    -> error: unknown /fold preset subcommand "huge" (balanced, loose, tight)
```

<div v-pre>

```yaml
# commands/fold.yaml (shipped, abridged)
version: 2.0.0
name: fold
usage:
  - args: "threshold <n|default>"
    help: "Tokens since the last checkpoint that trigger a fold (e.g. 2000, 2k)"
subcommands:
  "":                                  # bare /fold
    run:
      - tool: session_ops
        args: {operation: get, key: fold}
        print: "auto: {{.value}}, threshold: {{.threshold}} tokens"
  threshold:
    setting:
      key: fold-threshold              # a token count, or default
      show: "Fold threshold: {{.value}} tokens"
      set: "Fold threshold set to {{.value}} tokens (saved)"
      invalid: "threshold must be a positive token count (e.g. 2000, 2k) or default"
  preset:
    subcommands:
      tight:
        run:
          - tool: session_ops
            args: {operation: set, key: fold-threshold, value: "1000"}
            as: threshold              # named for the next step; not printed
          - tool: session_ops
            args: {operation: set, key: fold-items, value: "3"}
            print: "threshold={{.steps.threshold.value}}, items={{.value}}"
```

</div>

A tool step with `as:` and no `print:` prints nothing: its result is for the steps after it.

A `"*"` entry takes any first word no other entry names, and gets that word as its first argument. `/model` uses it: `/model list` is the `list` entry, `/model gpt-4o` is the `"*"` entry switching to `gpt-4o`.

```yaml
# commands/model.yaml (shipped, abridged)
subcommands:
  "":                                   # bare /model: the picker
    run:
      - tool: session_ops
        args: {operation: model_pick}
  "*":                                  # /model <name or URL>
    run:
      - tool: session_ops
        args: {operation: model, name: "$1"}
  list:
    run:
      - tool: session_ops
        args: {operation: models}
        print: "..."                    # the table, from the JSON models returns
  hff:
    subcommands:
      search:
        run:
          - tool: hf_search             # a python: tool asset, also offered to the model
            args: {query: "$@"}
```

A long answer pages in an interactive terminal (Enter/Space for the next page, `q` to stop).

## See also

- [Agent mode](/agent/) - overview and starting the REPL
- [Built-in tools](/agent/tools) - tools available to the model

## Declared commands on the API

A declared command (a `commands/` asset with `run:` or `setting:`) is also served on the commands API under its own name. The call runs the same steps as `/name` in the REPL, against the active session, and returns each model reply in order. `command:` steps run in the REPL and print their output to the terminal rather than returning it.
