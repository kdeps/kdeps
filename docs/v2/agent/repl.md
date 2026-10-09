# Agent loop REPL features

Runtime behaviors of the interactive agent loop REPL - pasting, rendering, notifications, and the tool budget. This is agent mode only. For starting the loop and registering workflows as tools, see [Agent mode](/agent/); for the slash commands, see [REPL slash commands](/agent/commands).

## Pasting

Paste a block of text and the REPL treats it as one prompt, not one turn per
line - it uses the terminal's bracketed-paste mode, which the REPL re-enables
before every prompt so a child process (an editor opened with `!`, a pager) or a
terminal that resets it cannot leave a later paste splitting into per-line
submissions. Works in any modern terminal, tmux, and screen. What happens next
depends on the size:

```d2
direction: right
paste: "Paste" {shape: oval}
check: "one line\nAND <= 20 words\nAND <= 240 chars?" {shape: diamond}
inline: "Inline as literal,\neditable text"
stage: "Save to memory as paste:<id>\nline + history show @paste:<id>"
model: "Model receives the\nfull text on submit" {shape: oval}
paste -> check
check -> inline: yes
check -> stage: no
inline -> model
stage -> model: "@paste:<id> expands\nback to the text"
```

A **small** single-line paste is inserted as ordinary editable text on the input line.

Any other paste is saved to memory under an id like `paste:3f9a1c2e` (in the
folder's memory under `~/.kdeps/memory/`). A multi-line paste of up to 4 lines,
20 words and 240 characters is shown verbatim on the edit line; a larger one is
shown as `@paste:3f9a1c2e` so the prompt and your scrollback stay readable.
Press Enter once to submit: the reference expands back to the pasted text, so
the model always gets the whole paste.

```text
you> why does this fail: @paste:3f9a1c2e      <- what the line and history keep
model sees: why does this fail:
--- @paste:3f9a1c2e ---
<the full pasted text>
```

REPL history (Up arrow, `Ctrl+R`) stores the `@paste:<id>` reference, not the
text. Recalling it in a later session still works, because the paste lives in
the folder's memory.

The paste is a single character on the edit line, so you can **edit around
it**: use the arrow keys (or `Ctrl+A` / `Ctrl+E`) to move before or after it
and type there - for example paste a stack trace and type
`why does this happen: ` in front of it, then submit.

## Line editing and history search

The prompt line supports standard readline editing: `Ctrl+A`/`Ctrl+E` to jump to the start/end of the line, `Ctrl+U` to clear before the cursor, `Ctrl+K` to clear after it, and Up/Down to step through prompt history.

`Ctrl+R` starts an incremental backward search through history - type any substring and the line fills in with the most recent match, narrowing as you type more; press `Ctrl+R` again to jump to the next older match. `Ctrl+S` searches forward the same way, walking back toward more recent matches - useful when `Ctrl+R` has stepped past the one you wanted. Press Enter to run the matched line, or `Ctrl+G`/Escape to cancel and return to what you were typing.

`Ctrl+S` is intercepted by some terminals or multiplexers for flow control (XON/XOFF) before it reaches the REPL; if it does nothing for you, that is a terminal setting, not kdeps - `stty -ixon` in your shell config re-enables it.

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

## Memory references

`@<memory-id>` expands to that memory entry's value, the same way `@notes.txt`
expands to the file's contents. Any memory id works: pastes (`paste:<id>`),
entries the model saved with `memory_save`, or ids listed by `/memory`. Tab
completes memory ids after `@` alongside file paths.

```bash
# Reuse an earlier paste
compare @paste:3f9a1c2e with the output of the last run

# Pull a saved memory entry into the prompt
follow @plan:migration step by step
```

The model receives:

```text
compare
--- @paste:3f9a1c2e ---
<the pasted text>
with the output of the last run
```

An existing file takes priority over a memory id with the same name. An unknown
id is left unchanged in the text.

## Prompt refinement

Before each turn, kdeps runs one cheap LLM call that rewrites a terse or
under-specified prompt into a clearer, self-contained version and runs the turn
on the rewrite. It is on by default; when the prompt is changed the REPL
prints `[refine] -> <rewritten prompt>` before any work starts, and the rewrite
is what gets saved to the session. Toggle with `/refine on|off` (persists to
`~/.kdeps/agent-loop-settings.yaml`). Full details: [Prompt refinement](/agent/refine).

## Response rendering

The REPL renders the model's markdown responses - headings, bold, lists, tables, and syntax-highlighted code blocks - in color. It auto-detects the terminal's color depth (truecolor, 256-color, or none) and downsamples the palette to match, so colors render correctly on terminals without 24-bit color (e.g. macOS Terminal.app) instead of collapsing to gray. Output piped to a file is left uncolored.

When extended reasoning is enabled (`/thinking`), the streamed reasoning is rendered as live markdown, updating in place as tokens arrive, shown in muted gray beneath a `* thinking` header and behind a dim left gutter (`|`) so the whole block reads as a distinct aside from the final answer. Inline code renders styled (by color, not literal backticks) in both the reasoning and the response.

## Custom harness

Every piece of text kdeps sends the model to shape its *behavior* - not the conversation itself - is a YAML file too: tool-use rules, the sandbox-hallucination reinforcement, and the system prompts behind compaction, goal planning, judging, and prompt refinement. Together they are the "harness." Like themes, the built-ins are compiled in, and you can add your own by dropping a file into `~/.kdeps/harness/<name>.yaml`:

```yaml
# ~/.kdeps/harness/house-style.yaml
name: house-style        # optional - defaults to the filename without its extension
kind: preamble-section    # "preamble-section" (sent on every turn) or "standalone" (looked up by name)
order: 200                 # only meaningful for preamble-section; controls where it lands among the others
body: |-
  Always answer in one paragraph, then a bulleted "next steps" list.
```

A `preamble-section` entry is appended to every turn's system preamble automatically - no other configuration needed. A file that reuses a built-in's name (e.g. your own `safety.yaml`) replaces that section outright; this includes the safety/accuracy/honesty rules, so overriding one is possible but is your call, the same as a custom theme picking illegible colors.

The built-in `preamble-section` names are `memory`, `tools`, `narration`, `autonomy`, `safety`, `errors`, `scope`, `accuracy`, `honesty`, `code`, `output`, `internals`, and `use-kdeps-tools`. The built-in `standalone` names - looked up individually, not auto-assembled, so a brand-new `standalone` name from you has no effect unless it reuses one of these - are `m365-sandbox`, `tools-reminder`, `skills-preamble`, `compaction-system`, `compaction-user`, `compaction-update-user`, `goal-plan-system`, `goal-confirm-system`, `judge-roster-system`, `judge-system`, `refine-system`, `branch-summary`, and `handshake` (see [Session-integrity handshake](/agent/tools-handshake#session-integrity-handshake)).

There is no `/harness` command: unlike a theme, harness content loads once at startup and is not switched at runtime.

## Turn-complete alert

When a turn takes a while (a long research loop, a slow local model), the REPL rings the terminal and posts a desktop notification once the response is ready, so you can step away and come back when it beeps:

- The terminal bell marks the tab/window as having activity in most terminals, tmux, and screen.
- An **OSC 9** desktop notification (`kdeps: response ready`) appears in terminals that support it (iTerm2, WezTerm, kitty); it is silently ignored elsewhere.

Only turns longer than a threshold alert, so quick replies stay quiet.

| Env var | Effect |
|---------|--------|
| `KDEPS_NOTIFY=off` | Disable the alert entirely |
| `KDEPS_NOTIFY_MIN=<dur>` | Minimum turn duration to alert (default `10s`; `0` = every turn) |

## Auto-retry

Transient LLM errors (HTTP 429, 5xx, network timeouts) are automatically retried up to 3 times with exponential backoff (2s, 4s, 8s). Context-overflow and authentication errors are not retried.

## Tool budget and stall timeout

The agent loop tracks a tool budget (`MaxToolRounds`) that limits how many tool calls the agent can make per turn. When the budget is nearly exhausted, the REPL presents interactive options: `(i)ncrease` the budget (adds 100 rounds), `(c)hange` to a specific number (`0` = unlimited), or `(i)gnore` to continue. When `AutoToolAllocation` is enabled in config, the budget increases automatically without prompting.

When a tool stalls (no output for the stall-timeout duration), the default is to auto-increase the timeout by the increment (default 5m) and announce it. Set `/model tool set autokill on` to kill a stalled tool at the timeout instead (mutually exclusive with auto-increase). Tune both with `/model tool set rounds <n>` and `/model tool set stall-timeout <dur>`. `AutoToolAllocation` (budget) and `AutoStallAllocation` (stall time) are independent and both on by default.

## See also

| Topic | Page |
|---|---|
| Look, custom themes, the context-path status line | [Themes](/agent/repl-themes) |
| Token budget, compaction, fold | [Context window](/agent/repl-context) |
| Per-folder history and the resume picker | [Sessions](/agent/repl-sessions) |
| `/upgrade`, nightly, and a pinned version | [Updating kdeps](/agent/repl-updating) |

- [Agent mode](/agent/) - starting the loop, tool registration
- [REPL slash commands](/agent/commands) - the full command table
- [Skills and prompt templates](/agent/skills) - context files that teach the agent
- [Local model management](/agent/models) - `/model`, `/context`, running servers
