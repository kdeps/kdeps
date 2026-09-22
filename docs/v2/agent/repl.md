# Agent loop REPL features

Runtime behaviors of the interactive agent loop REPL - pasting, rendering, notifications, context size, sessions, and updates. This is **agent mode** only. For starting the loop and registering workflows as tools, see [Agent mode](/agent/); for the slash commands, see [REPL slash commands](/agent/commands).

## Pasting

Paste a block of text and the REPL treats it as **one prompt**, not one turn per
line - it uses the terminal's bracketed-paste mode, which the REPL re-enables
before every prompt so a child process (an editor opened with `!`, a pager) or a
terminal that resets it cannot leave a later paste splitting into per-line
submissions. Works in any modern terminal, tmux, and screen. What happens next
depends on the size:

```d2
direction: right
paste: "Paste" {shape: oval}
check: "<= 4 lines\nAND <= 20 words\nAND <= 240 chars?" {shape: diamond}
inline: "Inline as literal,\neditable text"
stage: "Write to a temp file\nshow [pasted N lines @path]"
model: "Model receives the\nfull text on submit" {shape: oval}
paste -> check
check -> inline: yes
check -> stage: no
inline -> model
stage -> model: "@path expands\nback to contents"
```

A **small** paste is inserted as ordinary editable text on the input line.

A **large** paste is staged to a file under a temp dir so the prompt and your
scrollback stay readable; the line shows
`[pasted 123 lines @/tmp/kdeps-paste-xxxx/paste-1.txt]`. Press Enter once to
submit: that marker is expanded back to the file's contents, so the model always
gets the whole paste - only the on-screen line and the REPL history keep the
short form. The temp dir is removed when the REPL exits.

The large-paste marker is a single character on the edit line, so you can **edit
around it**: use the arrow keys (or `Ctrl+A` / `Ctrl+E`) to move before or after
it and type there - for example stage a stack trace and type
`why does this happen: ` in front of it, then submit.

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

- Image/binary refs (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`, `.tiff`, `.pdf`, `.mp3`, `.mp4`, `.wav`) are sent as multimodal content to the LLM.
- Text file refs are expanded inline in the prompt.
- Unresolvable refs (file not found, access denied) are left unchanged in the text.

## Prompt refinement

Before each turn, kdeps runs one cheap LLM call that rewrites a terse or
under-specified prompt into a clearer, self-contained version and runs the turn
on the rewrite. It is **on by default**; when the prompt is changed the REPL
prints `[refine] -> <rewritten prompt>` before any work starts, and the rewrite
is what gets saved to the session. Toggle with `/refine on|off` (persists to
`~/.kdeps/agent-loop-settings.yaml`). Full details: [Prompt refinement](/agent/refine).

## Response rendering

The REPL renders the model's markdown responses - headings, bold, lists, tables, and syntax-highlighted code blocks - in color. It **auto-detects the terminal's color depth** (truecolor, 256-color, or none) and downsamples the palette to match, so colors render correctly on terminals without 24-bit color (e.g. macOS Terminal.app) instead of collapsing to gray. Output piped to a file is left uncolored.

When extended reasoning is enabled (`/thinking`), the streamed reasoning is rendered as **live markdown**, updating in place as tokens arrive, shown in muted gray beneath a `* thinking` header and behind a dim left gutter (`|`) so the whole block reads as a distinct aside from the final answer. Inline code renders styled (by color, not literal backticks) in both the reasoning and the response.

## Themes

`/theme <name>` (or `--theme <name>`, or `KDEPS_THEME=<name>`) changes the REPL's entire look - banner, prompt, the text you type, model name, streamed responses, thinking blocks, tool summaries. `normal` is the bright default; every other theme is a disguise that no longer reads as "an AI session on model X" to anyone glancing at your screen in a cafe, on a plane, or in an open office. There's no separate on/off flag - picking a theme takes effect immediately.

The `/model` picker, the `/settings` picker, and the startup resume picker (`--resume`/session history) run outside the REPL's own text rendering, in their own full-screen views - but they pick up the exact same theme colors (accent, success, warning, dim, bold), not just a muted/bright toggle. Switching to `vim` recolors those pickers with vim's yellow/green/red accents; switching to `black` collapses them to the same flat gray as everything else.

```bash
kdeps --theme black             # start disguised
```

```text
/theme              show the current theme and the list of valid names
/theme list         same as bare /theme - built-in and custom names shown separately
/theme vim          switch themes - normal, black, linux, vim, emacs, or a custom name
```

| Theme | Look |
|---|---|
| `normal` (default) | The bright default palette - no disguise |
| `black` | A single flat, legible dark gray (`#767676`) for every element - muted and monochrome, but readable, not near-invisible |
| `linux` | Plain, monochrome-ish light-gray-on-black, like a default terminal with no syntax highlighting |
| `vim` | vim's classic default colorscheme conventions (yellow keywords, cyan identifiers, red strings) and a `: ` command-line prompt |
| `emacs` | A common terminal-Emacs highlight set (purple keywords, blue functions, salmon strings) and an `M-x ` prompt |

`/theme <name>` writes `theme: <name>` to `~/.kdeps/agent-loop-settings.yaml`, so the next `kdeps` starts with it too. Precedence: `--theme` flag, then `KDEPS_THEME`, then the persisted setting, then `normal`.

Every theme but `normal` renders the model-name color at full legibility, so the literal model name is shortened to initials in the status line by default - `claude-sonnet-5` becomes `CS5`, `llama3.2:1b` becomes `L21` - hidden by content, not color. Override this with `/model name`:

```text
/model name              show the current mode
/model name show         always show the literal model name, regardless of theme
/model name hide         omit the model name from the modeline entirely
/model name abbreviate   always abbreviate, even under normal
/model name auto         back to the theme-based default (the factory setting)
```

`/model name <mode>` persists to `~/.kdeps/agent-loop-settings.yaml` the same way `/theme` does. The spinner that appears while waiting for a response never carries a descriptive word in any theme - no "generating," no "thinking" - just the animated glyph and the token counter.

### Context-path status line

Directly above the session counter, whenever the running model's context window is known, a line shows what this turn's prompt is made of:

```
[turn 12.4k/200k | sys 8.1k | mem 1.2k | hist 2.4k | bash_exec 700]
[sent 1.2m | generated 30]
```

`turn A/B` is this prompt's size over the model's context window. The groups after it are that same prompt, split: `sys` system prompt, `mem` memory, `hist` conversation history, `goal` the active task, and one entry per tool name. Repeated calls to the same tool add to that one number. Past 5 groups the oldest collapse into `+N`. The line resets with each new prompt.

`sent` is the running total of prompt tokens handed to the model this session. History is sent again on every round, so `sent` grows faster than one turn's window and is not "how full the context is". `generated` is the running total of tokens the model wrote back. `web`, `sh`, `file`, and `src` appear beside them only after a call in that budget, as `used/limit`.

### Custom themes

Every theme - built-in or not - is a YAML file. Drop your own into `~/.kdeps/themes/<name>.yaml` and it shows up in `/theme`'s list immediately:

```yaml
# ~/.kdeps/themes/solarized.yaml
name: solarized     # optional - defaults to the filename without its extension
prompt: "$ "         # the literal prompt text this theme renders
bold: true
palette:
  heading: "#b58900"
  text: "#839496"
  # any field you omit falls back to the "normal" theme's value, so a
  # custom theme can override just a couple of accent colors and leave
  # everything else alone
```

A custom theme can reuse a built-in's name (e.g. your own `vim.yaml`) to override it. An invalid color value in one field is dropped with a warning; the rest of the file still loads.

Every `palette:` field is optional; the full set is `heading`, `link`, `code`, `codeBlock`, `text`, `thinking`, `muted`, `bullet`, `quote`, `borderHr`, `synKeyword`, `synFunc`, `synStr`, `synComment`, `synNum`, `synType`, `synOp`, `replError`, `replMeta`, `replHeading`, `replSuccess`, `replPrompt`, `replInfo`, `replDim`, `bannerText`, `bannerBorder`, `modelsReady`, `modelsNoKey`, `modelsCurrent`, and `modelName` (the model name shown in the status line).

## Custom harness

Every piece of text kdeps sends the model to shape its *behavior* - not the conversation itself - is a YAML file too: tool-use rules, the sandbox-hallucination reinforcement, and the system prompts behind compaction, goal planning, judging, and prompt refinement. Together they're the "harness." Like themes, the built-ins are compiled in, and you can add your own by dropping a file into `~/.kdeps/harness/<name>.yaml`:

```yaml
# ~/.kdeps/harness/house-style.yaml
name: house-style        # optional - defaults to the filename without its extension
kind: preamble-section    # "preamble-section" (sent on every turn) or "standalone" (looked up by name)
order: 200                 # only meaningful for preamble-section; controls where it lands among the others
body: |-
  Always answer in one paragraph, then a bulleted "next steps" list.
```

A `preamble-section` entry is appended to every turn's system preamble automatically - no other configuration needed. A file that reuses a built-in's name (e.g. your own `safety.yaml`) replaces that section outright; this includes the safety/accuracy/honesty rules, so overriding one is possible but is your call, the same as a custom theme picking illegible colors.

The built-in `preamble-section` names are `memory`, `tools`, `narration`, `autonomy`, `safety`, `errors`, `scope`, `accuracy`, `honesty`, `code`, `output`, `internals`, and `use-kdeps-tools`. The built-in `standalone` names - looked up individually, not auto-assembled, so a brand-new `standalone` name from you has no effect unless it reuses one of these - are `m365-sandbox`, `tools-reminder`, `skills-preamble`, `compaction-system`, `compaction-user`, `compaction-update-user`, `goal-plan-system`, `goal-confirm-system`, `judge-roster-system`, `judge-system`, `refine-system`, `branch-summary`, and `handshake` (see [Session-integrity handshake](/agent/tools#session-integrity-handshake)).

There's no `/harness` command: unlike a theme, harness content loads once at startup and isn't switched at runtime.

## Turn-complete alert

When a turn takes a while (a long research loop, a slow local model), the REPL rings the terminal and posts a desktop notification once the response is ready, so you can step away and come back when it beeps:

- The terminal **bell** marks the tab/window as having activity in most terminals, tmux, and screen.
- An **OSC 9** desktop notification (`kdeps: response ready`) appears in terminals that support it (iTerm2, WezTerm, kitty); it is silently ignored elsewhere.

Only turns longer than a threshold alert, so quick replies stay quiet.

| Env var | Effect |
|---------|--------|
| `KDEPS_NOTIFY=off` | Disable the alert entirely |
| `KDEPS_NOTIFY_MIN=<dur>` | Minimum turn duration to alert (default `10s`; `0` = every turn) |

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

Set the default at startup with the `KDEPS_GGUF_CTX_SIZE` (gguf) or `KDEPS_LLAMAFILE_CTX_SIZE` (file) environment variables. In resource YAML, `contextSize:` on a `chat:` block overrides per call; for Ollama only, `ollamaNumCtx:` is also accepted and takes precedence.

### Text-embedded tool calls

Most models call tools through the backend's tool-use channel. Some instead
write the call as text --- `<tool_call>{"name":...,"arguments":{...}}</tool_call>`,
`<function=name>{...}</function>`, `<invoke name="..."><parameter name="...">...</parameter></invoke>`,
or a bare JSON object --- and sometimes follow it with a **self-written
`<tool_response>`** block and a false "done".

kdeps recovers a text-written tool call and runs it for real. A model-authored
`<tool_response>` is always a hallucination (only the runtime produces tool
results): kdeps strips it, does not accept the turn as finished, and nudges the
model once to make the actual call and wait for the real result. These markers
never reach the visible answer.

### In-turn tool history

A single turn can make many tool calls (`/model tool set rounds <n>`, default 200). The transcript of those calls and their results is re-sent to the model on every round, so a long tool loop can otherwise grow the request without bound - `/compact` and the auto-compaction that runs between turns do not fire mid-turn.

kdeps caps the in-flight transcript at roughly 24K tokens. Past that, the oldest complete tool round-trips are dropped (the leading conversation and this turn's question are always kept), and long tool-call arguments on the older kept steps are replaced with a `[N chars omitted]` placeholder. When this happens you see:

```
[context] trimmed 3 old tool step(s) to fit the window
```

The model keeps the most recent steps in full and a note that earlier ones were dropped. Tool *results* are separately capped at 16 KB each before they ever enter the transcript.

### Compacting the saved conversation

Only the user prompt and the final answer of each turn are saved to the session
- a turn's own tool transcript is never persisted. `/compact` summarizes the
older saved turns and keeps the last few verbatim; it runs whenever the session
has more than a handful of turns, regardless of size. Auto-compaction between
turns fires only once the saved conversation exceeds the token budget (`/model
tool set compact-threshold <n>`), which for a long run of short-but-tool-heavy
turns may never happen - use `/compact` directly. Neither affects a running
turn's tool output; that is bounded by the in-flight window above.

Both the compact budget (how much recent conversation stays verbatim) and the
auto-compaction threshold default to 3/4 of the model's known context window,
not a flat token count - a session on a small local model and one on a
200K-token cloud model each get a sensible default without you having to set
`/model tool set compact-budget <n>` yourself. Switching models with `/model`
recomputes both immediately for the new model; an unrecognized model keeps a
conservative flat default.

The conversation text being summarized is never re-evaluated as a kdeps
expression or template, even if it contains double-curly-brace syntax (kdeps
expressions you were discussing, another template language's tags, or
anything else that looks like one) - it reaches the summarizer exactly as
written.

### Fold

`/fold` is a lighter, more frequent cousin of `/compact` - instead of waiting
for the full compact-threshold safety net, it checkpoints the conversation
(same structured Goal/Progress/Decisions summary, saved to
[memory](/agent/memory-internals#checkpoint-summaries)) every time a
configurable amount of *new* conversation has accumulated since the last
checkpoint. It's on by default (2000 tokens, keeping the last 5 checkpoints
in the prompt's memory block) and needs no setup.

| Command | Effect |
|---|---|
| `/fold` | Show status: auto on/off, threshold, items cap, tokens accumulated since the last checkpoint |
| `/fold now` | Force a fold immediately, regardless of the threshold or the compaction budget |
| `/fold auto` / `/fold auto off` | Toggle automatic folding (default: on) |
| `/fold threshold <n>` | Token delta since the last checkpoint that triggers a fold (e.g. `4k`) |
| `/fold items <n>` | How many recent checkpoints stay in the active memory-prompt window |
| `/fold preset tight\|balanced\|loose` | Named threshold+items bundles - `balanced` is the shipped default |

All four settings persist across sessions, the same way `/model tool set
compact-threshold`/`compact-budget` already do.

`/fold now` is a *forced* fold: it ignores the threshold and the compaction
budget entirely and always summarizes everything except the last few turns,
same as `/compact`. The only way it comes back with nothing is too few turns
to have anything beyond what it always keeps verbatim - it then prints the
turn count needed and the current `in:`/`out:` token counter, a number to
check instead of just an assertion.

Both `/compact` and `/fold` refresh the cumulative token counter (`in:`/`out:`
in the status line) immediately after they run, since the summarization call
itself uses real tokens that would otherwise never get counted.

## Sessions

Conversations and the agent's memory are stored in `~/.kdeps/`, **partitioned by
the directory you launch `kdeps` from** - `~/.kdeps/sessions/` and
`~/.kdeps/memory/` each hold a subfolder per working directory. `cd` into a
project and `kdeps` sees that folder's history; a different folder starts clean.
Nothing is written into the project directory itself.

When you start `kdeps` in a folder that has saved sessions - and you did not pass
`--resume` or `--new` - a picker lists them:

```d2
direction: down
start: "kdeps  (no --resume, no --new)" {shape: oval}
has: "this folder has saved sessions?" {shape: diamond}
picker: "picker: each session's first prompt,\nturn count, last-active time\n+ 'Start a new session'"
resume: "chosen session's history restored" {shape: oval}
fresh: "clean session" {shape: oval}
start -> has
has -> picker: "yes (interactive)"
has -> fresh: "no / piped input"
picker -> resume: "pick a session"
picker -> fresh: "'Start a new session' / esc"
```

```text
kdeps                     # in a folder with history -> resume picker
kdeps --new               # skip the picker, start a clean session
kdeps --resume <id>       # resume a specific session directly (no picker)
```

- The picker only appears in an interactive terminal with at least one saved
  session for this folder and no `--resume` / `--new`. Piped input starts clean.
- Resuming, continuing, then exiting **updates the same session** - it does not
  fork a new one. The session is also saved after every turn, so a crash or
  kill still leaves it in the picker.
- Settings (`/refine`, `/theme`, the default model, tool tuning), the model
  cache, and everything else under `~/.kdeps/` are unchanged.
- `ctrl+d` on a highlighted session deletes it in place (the "Start a new
  session" row can't be deleted) - same as `/session delete <id>`, just
  reachable from the picker itself.
- Multiple `kdeps` instances can point at the same folder at once, each with
  its own picker and its own chosen (or new) session - the underlying session
  file is opened only for the moment of each read/write, not held open for an
  instance's whole run, so instances don't lock each other out.

```
/session list                  # this folder's saved sessions
/session save [name]           # save a named snapshot of the current session
/session load <id>             # restore a saved session
/session delete <id>           # delete a saved session
/session checkpoint            # print the current entry ID (for /session goto)
/session goto <entry-id>       # restore session to the turn at that entry ID
/session branches              # list stashed (pruned) turns from prior /session goto calls
/session import <path>         # load a JSONL session file exported from another run
```

`/session goto` is non-destructive: the pruned tail is stashed. Use `/session branches` to see stashed entry IDs, then `/session goto <id>` again to navigate back.

## Auto-retry

Transient LLM errors (HTTP 429, 5xx, network timeouts) are automatically retried up to 3 times with exponential backoff (2s, 4s, 8s). Context-overflow and authentication errors are not retried.

## Tool budget and stall timeout

The agent loop tracks a tool budget (`MaxToolRounds`) that limits how many tool calls the agent can make per turn. When the budget is nearly exhausted, the REPL presents interactive options: `(i)ncrease` the budget (adds 100 rounds), `(c)hange` to a specific number (`0` = unlimited), or `(i)gnore` to continue. When `AutoToolAllocation` is enabled in config, the budget increases automatically without prompting.

When a tool stalls (no output for the stall-timeout duration), the default is to **auto-increase** the timeout by the increment (default 5m) and announce it. Set `/model tool set autokill on` to **kill** a stalled tool at the timeout instead (mutually exclusive with auto-increase). Tune both with `/model tool set rounds <n>` and `/model tool set stall-timeout <dur>`. `AutoToolAllocation` (budget) and `AutoStallAllocation` (stall time) are independent and both on by default.

## Updating kdeps

kdeps checks GitHub for a newer stable release at startup (throttled to once every 24 hours, cached at `~/.kdeps/update-check.json`, bounded to 3 seconds) and, if one exists, prints a one-line notice under the banner:

```
Update available: v2.8.0 -> v2.9.0. Run /upgrade to update.
```

Run `/upgrade` (or `kdeps --upgrade` outside the REPL) any time to check immediately and install. What happens next depends on how kdeps was installed:

- **Homebrew** (`brew install kdeps/tap/kdeps`): prints `brew upgrade kdeps` instead of touching the binary - self-replacing it would desync Homebrew's own bookkeeping.
- **.deb / .apk package**: prints the matching package-manager upgrade command.
- **Standalone** (the `curl | sh` installer, or a manually downloaded binary): after a `[Y/n]` confirmation (skippable with `KDEPS_YES=1`), downloads the release archive for your platform, verifies its SHA256 against `checksums.txt`, and atomically replaces the running binary. Restart kdeps afterward.

### Nightly builds

kdeps also cuts a nightly build from `main` most days. `/upgrade nightly` (or `kdeps --upgrade --nightly`) switches the channel for that one check: it installs the latest nightly instead of the latest stable.

Nightly opt-in only works for a **standalone** install - Homebrew/.deb/.apk only ever track stable, so on those `/upgrade nightly` prints standalone-install instructions instead of a package-manager command. "Already up to date" for the nightly channel means you're running that exact nightly tag: a nightly reuses the current stable version number until the next stable release ships, so it is always offered until you are actually on it.

### Installing a specific version (including a downgrade)

`/upgrade <version>` (or `kdeps --upgrade --target-version <version>` outside the REPL) installs exactly that version, skipping the "is an update available" check entirely - the version was named explicitly, so kdeps installs it whether it's newer than the running build (upgrade), older (downgrade), or the same (reinstall):

```
/upgrade 2.35.0
Downgrade to v2.35.0 now? [Y/n]
```

Same standalone-only restriction as nightly: Homebrew/.deb/.apk can't be told to install a specific (especially older) version, so those print a link to the release page instead.

## See also

- [Agent mode](/agent/) - starting the loop, tool registration
- [REPL slash commands](/agent/commands) - the full command table
- [Skills and prompt templates](/agent/skills) - context files that teach the agent
- [Local model management](/agent/models) - `/model`, `/context`, running servers
