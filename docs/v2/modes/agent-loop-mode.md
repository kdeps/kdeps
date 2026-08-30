# Agent Loop Mode

Agent loop mode starts an interactive LLM REPL where whole workflows and components are registered as callable tools. The LLM decides which tool to invoke based on the user's prompt. Workflow tools run the full pipeline atomically so all `requires:` dependencies resolve correctly.

Running `kdeps` with no arguments starts a bare REPL with no workflow tools -- built-in tools (web_search, bash_exec, file ops, memory, etc.) are still available. Pass a path to also load workflows/agencies as tools.

## Starting the agent loop

```bash
kdeps                              # runs the agent loop REPL
kdeps --model llama3.2 --system "You are a DevOps assistant."  # override model/system prompt
kdeps --skill ~/.kdeps/skills/     # load skill files
kdeps --resume <session-id>        # continue a saved session
```

Pass a workflow or agency path to register it as a tool -- see [Single workflow vs folder](#single-workflow-vs-folder) below.

## REPL slash commands

Every REPL interaction is driven by slash commands -- `/model`, `/session`, `/thinking`, `/goal`, `/judges`, `/memory`, `/turo`, and more -- plus auto-detected shell commands and file references you can just type in plain, quoted English (`can you check "df -h"?`) instead of `!df -h`.

See [REPL Slash Commands](/modes/agent-loop-commands) for the full command table and how auto-detection works.

## Goal-directed execution

Every prompt becomes an explicit task list that Go code drives to completion -- the loop walks a cursor through the list that only ever moves forward, so a model cannot circle back over finished work or stall on a task until a budget expires. A task closes by calling `task_complete`/`task_fail`, optionally gated on real evidence (`RequireTaskEvidence`), and unproductive rounds escalate through re-anchor, narrow, force-close, and fail-forward so a goal always terminates.

See [Goal-Directed Execution](/modes/agent-loop-goals) for the full mechanics, including adaptive per-category tool budgets.

## Judge panel

Goal enforcement checks that the loop keeps moving; it says nothing about whether the final answer is actually *right*. A judge panel is an independent review of that answer, run after the turn produces it -- one or more reviewer personas, each with real tool access, check the output and can send it back for revision before it reaches you. `AutoJudges` is on by default in the interactive REPL; toggle or hand-configure the roster with `/judges`.

See [Judge Panel](/modes/agent-loop-judges) for the full review flow and roster configuration.

## Prompt reduction (turo)

`turo` is an optional token reducer. When the `turo` binary is on `PATH`, kdeps pipes everything it sends to the LLM through it first -- system preamble, input, tool results, and conversation history -- through a five-stage pipeline (filler deletion, defmatch, gloss swap, synonym swap, reduction) plus an arrow-connective pass. Code, file paths, and identifiers are preserved verbatim; if a reduction isn't smaller, the original passes through unchanged. Applies to agent mode only.

See [Prompt Reduction (turo)](/modes/agent-loop-turo) for the full pipeline, `/turo` controls, and install-time environment variables.

## Local model management

`/model <name>` switches models mid-session, downloading and starting a local server (`file`/`gguf`) if needed with a progress display so the first prompt never hits a cold-start error. `--model auto` routes across your configured `llm.models`; `--model auto-router` skips config entirely and always auto-discovers the best local or cloud fit. `/model hff` searches and downloads GGUF models from HuggingFace, `/model favorite` pins ones you use often, and `/model ps` manages running local servers.

See [Local Model Management](/modes/agent-loop-models) for the full reference: alias collisions across backends, registering a model by URL, and how a model is picked when none is configured.

## Updating kdeps

kdeps checks GitHub for a newer stable release at startup (throttled to once
every 24 hours, cached at `~/.kdeps/update-check.json`, and bounded to 3
seconds so a slow network never delays startup) and, if one exists, prints a
one-line notice under the banner:

```
Update available: v2.8.0 -> v2.9.0. Run /upgrade to update.
```

Run `/upgrade` any time to check immediately (always live, ignoring the
cache) and, if an update is available, install it:

```
/upgrade
```

What happens next depends on how kdeps was installed:

- **Homebrew** (`brew install kdeps/tap/kdeps`): prints `brew upgrade kdeps`
  instead of touching the binary -- self-replacing it would desync
  Homebrew's own bookkeeping.
- **.deb/.apk package**: prints the matching package-manager upgrade command.
- **Standalone** (the `curl | sh` installer, or a manually downloaded
  binary): after a `[Y/n]` confirmation (skippable with `KDEPS_YES=1`),
  downloads the release archive for your platform, verifies its SHA256
  against the release's `checksums.txt`, and atomically replaces the running
  binary. Restart kdeps afterward to use the new version.

The same flow is available without starting the REPL:

```bash
kdeps --upgrade
```

### Nightly builds

kdeps also cuts a nightly build from `main` most days. `/upgrade nightly`
(or `kdeps --upgrade --nightly` outside the REPL) switches the channel for
that one check: instead of the latest stable release, it checks and
installs the latest nightly.

```
/upgrade nightly
```

```bash
kdeps --upgrade --nightly
```

Nightly opt-in only works for a **standalone** install -- Homebrew/.deb/.apk
only ever track stable, so on those install methods `/upgrade nightly`
prints instructions for a standalone install instead of a package-manager
command (which would silently install a stable build, not the nightly you
asked for). "Already up to date" for the nightly channel means you're
running that exact nightly tag, not a newer-by-semver comparison -- a
nightly tag reuses the current stable version number until the next stable
release ships, so it's always offered until you're actually on it.

## Context window size

`/context` shows or changes the context window size for the current model. The effect depends on the backend:

| Backend | Effect |
|---------|--------|
| `file` (llamafile) | Kills the running server and restarts it with `--ctx-size <n>` |
| `gguf` (llama-server) | Kills the running server and restarts it with `--ctx-size <n>` |
| `ollama` | Sets `num_ctx` on the next request - no restart needed |
| Cloud (openai, anthropic, etc.) | No effect - context size is managed server-side |

```
/context              # show current size (e.g. "Context window: 4096 tokens")
/context 32768        # set to 32K
/context 128k         # shorthand - equivalent to 131072
```

You can also set the default at startup with the `KDEPS_GGUF_CTX_SIZE` (gguf) or `KDEPS_LLAMAFILE_CTX_SIZE` (file) environment variables.

In resource YAML, set `contextSize:` on any `chat:` block to override per-call:

```yaml
resources:
  - action: analyze
    chat:
      model: llama3.2
      contextSize: 32768   # restarts the server with this size if backend is file/gguf
      prompt: |
        Summarize: $request.body.text
```

For Ollama only, `ollamaNumCtx:` is also accepted and takes precedence over `contextSize:`.

## Turn-complete alert

When a turn takes a while (a long research loop, a slow local model), the REPL rings the terminal and posts a desktop notification once the response is ready, so you can step away and come back when it beeps:

- The terminal **bell** marks the tab/window as having activity in most terminals, tmux, and screen.
- An **OSC 9** desktop notification (`kdeps: response ready`) appears in terminals that support it (iTerm2, WezTerm, kitty); it is silently ignored elsewhere.

Only turns longer than a threshold alert, so quick replies stay quiet.

| Env var | Effect |
|---------|--------|
| `KDEPS_NOTIFY=off` | Disable the alert entirely |
| `KDEPS_NOTIFY_MIN=<dur>` | Minimum turn duration to alert (default `10s`; `0` = every turn) |

## Pasting multiple lines

Paste a block of text and the REPL treats it as **one prompt**, not one turn per line. The whole block collapses to a single `▧` marker on the input line (its full content is kept off-screen so a large paste never redraws the terminal); press Enter once to submit and the marker is replaced by the full pasted text, with embedded newlines preserved. This uses the terminal's bracketed-paste mode, so it works in any modern terminal, tmux, and screen.

Because the paste is a single character on the edit line, you can **edit around it**: use the arrow keys (or `Ctrl+A` / `Ctrl+E` for line start/end) to move before or after the `▧` and type text there — for example paste a stack trace and type `why does this happen: ` in front of it, then submit. Everything you type around the marker is preserved and sent with the paste as one prompt.

## Response rendering

The REPL renders the model's markdown responses — headings, bold, lists, tables, and syntax-highlighted code blocks — in color. It **auto-detects the terminal's color depth** (truecolor, 256-color, or none) and downsamples the palette to match, so colors render correctly on terminals without 24-bit color (e.g. macOS Terminal.app) instead of collapsing to gray. Output piped to a file is left uncolored.

When extended reasoning is enabled (`/thinking`), the streamed reasoning is rendered as **live markdown**, updating in place as tokens arrive, shown in muted gray beneath a `* thinking` header and behind a dim left gutter (`│`) so the whole block reads as a distinct aside from the final answer. Inline code renders styled (by color, not literal backticks) in both the reasoning and the response.

## Built-in tools

The agent has access to a set of built-in tools the LLM can call without any YAML configuration -- file operations, shell execution, web/search, memory, SQL, embeddings, and more -- plus name aliases so a model that calls `grep` or `cat` still reaches the right tool. Tools requiring credentials only register when the matching environment variable is set.

See [Built-in Tools](/modes/agent-loop-tools) for the full catalog, permission modes, lean mode, and agent presets.

## Agent registries

The agent loop maintains three in-memory registries for lifecycle management -- TaskRegistry, TeamRegistry, and CronRegistry -- each with its own set of LLM-facing tools (`task_*`, `team_*`, `cron_*`) for creating, tracking, and coordinating work across turns and sessions.

See [Agent Registries](/modes/agent-loop-registries) for the full method and tool reference.

## Approval tokens

When a tool call is denied by the permission mode, the agent can request a one-time exception via an approval token (`approval_request`/`approval_grant`/`approval_list`/`approval_revoke`) -- a scoped override for a specific tool+action combination without relaxing the overall permission mode.

See [Approval Tokens](/modes/agent-loop-approvals) for the full lifecycle and an example flow.

## Multimodal input

Attach images and other binary files to your prompt using `@`:

```bash
# Attach a local image
describe @photo.png what is in this image?

# Attach multiple images
compare @before.jpg @after.jpg what changed?

# Attach a remote image URL
analyze @https://example.com/chart.png what trend does this show?

# Embed a text file inline (text files expand inline, not as attachments)
review @notes.txt and summarize the key points
```

- Image/binary refs (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`, `.tiff`, `.pdf`, `.mp3`, `.mp4`, `.wav`) are sent as multimodal content to the LLM
- Text file refs are expanded inline in the prompt
- Unresolvable refs (file not found, access denied) are left unchanged in the text

## Skills

Skills are markdown files with optional YAML frontmatter that teach the agent how to behave in specific contexts. Place them in `~/.kdeps/skills/` or pass `--skill <path>` at startup.

```markdown
---
name: code-review
description: Guidelines for reviewing Go code
---

Always check for error handling. Prefer early returns over nested conditions.
```

Skills are discovered from:
- `~/.kdeps/skills/` (global)
- `./.kdeps/skills/` (project-local)
- Paths passed with `--skill` (explicit, repeatable)

Invoke a skill from the REPL with `/<skill-name>` or `/<skill-name> extra context here`.

**Progressive disclosure (token cost):** the system prompt lists only each skill's name and description - never the full body. Skill instructions are re-sent on every LLM call as part of the system prompt, so embedding full bodies for a large skill set would burn tokens every turn. Instead, the agent calls the built-in `load_skill` tool with a skill name to pull that skill's full instructions on demand, only when a task actually needs it.

**Related skills:** when skills are (re)loaded, kdeps builds a small [kartographer](https://github.com/kdeps/kartographer) reference/topic graph over each skill-library root -- the same mechanism `codeIntelligence`'s [`indexFolder`/`graphFile`](../resources/codeintelligence-graph) uses on any folder. A skill is related to another if its `SKILL.md` links to it (`[other](../other/SKILL.md)`), or if both declare the same `topics:`/`tags:` in frontmatter:

```markdown
---
name: code-review
description: Guidelines for reviewing Go code
topics: [go, quality]
---

Always check for error handling. See [testing](../testing/SKILL.md) for coverage expectations.
```

When `load_skill` returns a skill whose graph has related skills, it appends a hint listing their names so the model can decide whether to load them too, instead of guessing skill names cold. This is purely additive -- a skill with no links or topics behaves exactly as before.

## Prompt templates

Prompt templates are reusable named prompts loaded from `.md` files. They work exactly like skills: invoke them by name from the REPL.

```markdown
---
name: review-pr
description: Review a GitHub pull request
argument-hint: <PR number or URL>
---

Review the pull request at $1. Check for: correctness, test coverage, and breaking changes.
```

Place templates in `~/.kdeps/prompts/` or `./.kdeps/prompts/`. Templates use the same placeholder syntax as skills: `$1`, `$2`, `$@`, `${1:-default}`.

```bash
/review-pr 1234
/summarize this document for a technical audience
```

## Instructions

The agent automatically discovers instruction files by walking up the directory tree from CWD:

- `CLAUDE.md`, `CLAUDE.local.md` at any ancestor directory
- `.kdeps/CLAUDE.md`, `.kdeps/instructions.md` at any ancestor directory

Duplicate content (by hash) is deduplicated. Total injected context is capped at ~12 KB. Instructions are injected into the system prompt at startup.

## Session persistence

Every conversation is saved as a JSONL file under `~/.kdeps/sessions/`. To resume a previous session:

```bash
kdeps --resume <session-id>
```

Session IDs are shown at the start of each run.

### Session commands

```
/session list                  # list all saved sessions
/session save [name]           # save current session
/session load <id>             # restore a saved session
/session delete <id>           # delete a saved session
/session checkpoint            # print the current entry ID (for /session goto)
/session goto <entry-id>       # restore session to the turn at that entry ID
/session branches              # list stashed (pruned) turns from prior /session goto calls
/session import <path>         # load a JSONL session file exported from another run
```

`/session goto` is non-destructive: the pruned tail is stashed. Use `/session branches` to see stashed entry IDs, then `/session goto <id>` again to navigate back.

### Auto-retry

Transient LLM errors (HTTP 429, 5xx, network timeouts) are automatically retried up to 3 times with exponential backoff (2s, 4s, 8s). Context-overflow and authentication errors are not retried.

### Tool budget and stall timeout

The agent loop tracks a tool budget (`MaxToolRounds`) that limits how many tool calls the agent can make per turn. When the budget is nearly exhausted, the REPL presents interactive options: `(i)ncrease` the budget (adds 100 rounds), `(c)hange` to a specific number (`0` = unlimited), or `(g)nore` to continue with the current budget. When `AutoToolAllocation` is enabled in config, the budget increases automatically without prompting.

Similarly, when a tool stalls (no output for the stall timeout duration), the default is to **auto-increase** the timeout by the increment (default 5m) and announce it. Set `/model tool set autokill on` to **kill** a stalled tool at the timeout instead (mutually exclusive with auto-increase). Both are shown by `/model tool`. `AutoToolAllocation` (budget) and `AutoStallAllocation` (stall time) are independent and both on by default.

Both settings can also be tuned with `/model tool set rounds <n>` and `/model tool set stall-timeout <dur>`.

## Single workflow vs folder

```bash
kdeps ./my-agent/     # registers the workflow as an LLM-callable tool (named after metadata.name)
kdeps ./agents/       # registers every workflow and agency in the folder as a separate LLM-callable tool
```

When you point to a folder, kdeps discovers every workflow and agency file inside it (recursively). Each becomes a separate tool. The tool name is `metadata.name` from the workflow's manifest -- not the filename.

## Concrete example

Given this workflow:

```yaml
# my-agent/workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent          # this becomes the tool name the LLM sees
  version: "1.0.0"
  description: "Answers questions about our product"
  targetActionId: response

settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

Running:

```bash
kdeps ./my-agent/
```

The LLM receives one tool named `my-agent`. When it calls that tool, kdeps runs the full workflow DAG -- every resource in dependency order -- and returns `apiResponse.response` to the LLM.

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

## Tool registration

Built-in tools (web_search, bash_exec, file ops, memory, etc.) are always registered regardless of path. Workflow/agency/component tools depend on what you point at:

| Target | Workflow/agency/component tools registered |
|--------|-----------------|
| No path | None |
| Single workflow file/dir | One tool (`metadata.name`) + one tool per component |
| Single agency file | One tool (`agency metadata.name`) |
| Folder | One tool per workflow/agency found recursively + component tools |

## Command

```bash
kdeps [path] [flags]
```

`[path]` is optional. When provided it must be a workflow/agency file or directory. The tool name comes from `metadata.name` -- not the filename.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `KDEPS_AGENT_MODEL` or `llama3.2` | LLM model name |
| `--backend` | `KDEPS_AGENT_BACKEND` or `file` | LLM backend (`file`, `gguf`, `ollama`, `openai`, ...) |
| `--base-url` | `KDEPS_AGENT_BASE_URL` | LLM API base URL |
| `--system` | (none) | System prompt injected at conversation start |
| `--skill` | (none) | Path to a skill file or directory (repeatable) |
| `--prompt` | (none) | Path to a prompt templates directory (repeatable) |
| `--resume` | (none) | Session ID to resume a previous conversation |
| `--debug` | false | Enable debug logging |

### Environment variables

```bash
KDEPS_AGENT_MODEL=llama3.2
KDEPS_AGENT_BACKEND=file              # default: local llamafile
# KDEPS_AGENT_BACKEND=gguf           # llama.cpp via llama-server
# KDEPS_AGENT_BACKEND=ollama         # requires ollama server
# KDEPS_AGENT_BASE_URL=http://localhost:11434
```

## Examples

```bash
# Bare REPL, no workflow tools
kdeps

# Advanced usage: point at a workflow or agency directory to register it as a tool
# Registers the workflow as an LLM-callable tool
kdeps ./my-agent/

# Registers every workflow in the folder as an LLM-callable tool
kdeps ./agents/

# Override model and system prompt
kdeps ./agents/ --model mistral --system "You are a data analyst."

# GGUF backend with local model file
kdeps --backend gguf --model qwen3.5-4b

# OpenAI backend
KDEPS_AGENT_BACKEND=openai kdeps ./agents/ --model gpt-4o

# Load a skill directory
kdeps --skill ~/.kdeps/skills/

# Resume a previous session
kdeps --resume abc123def456
```

## Differences from workflow mode

| | Workflow mode (`kdeps run`) | Agent loop mode (`kdeps [path]`) |
|--|-----------------------------|---------------------------------|
| Execution | DAG, deterministic | LLM loop, tool-driven |
| Entry point | `metadata.targetActionId` | User prompt |
| Unit of work | Individual resources | Whole workflows |
| Tools exposed | N/A | One per workflow + one per component |
| Input | Single workflow path | Optional file or folder |
| Session memory | None | Multi-turn, persistent JSONL |

## See also

- [REPL Slash Commands](/modes/agent-loop-commands) - Full command reference
- [Built-in Tools](/modes/agent-loop-tools) - Tool catalog, permission modes, lean mode
- [Goal-Directed Execution](/modes/agent-loop-goals) - Task decomposition and enforcement
- [Judge Panel](/modes/agent-loop-judges) - Independent review of the final answer
- [Local Model Management](/modes/agent-loop-models) - Switching, auto-routing, running servers
- [Agent Registries](/modes/agent-loop-registries) - Task/team/cron tools
- [Approval Tokens](/modes/agent-loop-approvals) - One-time permission exceptions
- [Prompt Reduction (turo)](/modes/agent-loop-turo) - Optional token reducer
- [Workflow Mode](workflow-mode) - Deterministic DAG pipelines
- [LLM Provider Reference](/reference/llm-providers) - Backend config and model names
- [Agencies](/concepts/agency) - Multi-agent orchestration
