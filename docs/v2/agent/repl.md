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
folder's memory, a note in the project's `memory/` folder in the knowledge vault). A multi-line paste of up to 4 lines,
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

The prompt line supports standard readline editing: `Ctrl+A`/`Ctrl+E` to jump to the start/end of the line, `Ctrl+U` to clear before the cursor, `Ctrl+K` to clear after it, and Up/Down to step through prompt history. Those lines are stored in `~/.kdeps/repl_history`. The full-screen prompt (`KDEPS_TUI=1`) reads and writes the same file: `Ctrl+P` recalls the previous line and `Ctrl+N` the next. Typing, pasting, or deleting on a recalled line keeps that edit, so the next `Ctrl+N` does not replace it. A click in the main pane or the feed focuses that pane for the arrow keys. A click in the prompt moves the cursor. `Shift+Left` and `Shift+Right` select, and so does dragging across the prompt. `Ctrl+Shift+C` copies the selection and `Ctrl+Shift+X` cuts it, so `Ctrl+Y` can put it back. With no selection, `Ctrl+Shift+C` copies the visible main pane into that same buffer, as one line. `Ctrl+Shift+A` selects the whole line. Typing or Backspace replaces the selection. `Ctrl+Left` and `Ctrl+Right` move by word (`Alt+Left` and `Alt+Right`, or `Alt+B` and `Alt+F`, do the same). `Alt+D` deletes the next word. `Alt+Backspace` deletes the word before the cursor, the same as `Ctrl+W`. `Ctrl+Y` puts back the text removed by `Ctrl+K`, `Ctrl+U`, `Ctrl+W`, or `Alt+D`. Repeating one of those without typing in between joins the removed text into one yank. `Ctrl+O` opens that line in an external editor (`$KDEPS_EDITOR`, then `$VISUAL`, then `$EDITOR`, then `vi`). Saving replaces the prompt and does not send it. A newline becomes a space. A blank file clears the prompt. A failed editor leaves the prompt as it was. `Ctrl+D` deletes the character under the cursor and does not save it for `Ctrl+Y`. On an empty prompt it quits. `d` still types. A full-screen prompt longer than the row scrolls so the cursor stays on screen. A click lands on the letter that is drawn. Esc clears the selection and leaves the text. Esc while a recalled line is showing puts the unsent text back. It does not quit.

`Ctrl+R` starts an incremental backward search through history - type any substring and the line fills in with the most recent match, narrowing as you type more; press `Ctrl+R` again to jump to the next older match. `Ctrl+S` searches forward the same way, walking back toward more recent matches - useful when `Ctrl+R` has stepped past the one you wanted. Press Enter to run the matched line, or `Ctrl+G`/Escape to cancel and return to what you were typing.

`Ctrl+S` is intercepted by some terminals or multiplexers for flow control (XON/XOFF) before it reaches the REPL; if it does nothing for you, that is a terminal setting, not kdeps - `stty -ixon` in your shell config re-enables it.

## Full-screen scrolling

`KDEPS_TUI=1` draws the session on the alternate screen. Up and Down scroll the focused pane one row. PgUp and PgDown scroll a page. `Ctrl+Up` and `Ctrl+Down` scroll half a page. Home jumps to the oldest row and End returns to the newest. Ctrl+End follows the main pane even when the feed is focused. The offset stops at either end, so one PgDown from the top moves forward. While the main pane is on an older row, the status row says `not following` and new output leaves those rows in place. End, or scrolling back to the newest row, follows again. Sending a prompt follows immediately. Ctrl+Space folds the output at the bottom of the main pane down to its first line, and folds it open again. A one-line output stays as it is. Space still types. Ctrl+Z suspends the full-screen view and returns to the shell. The view comes back when that shell exits. z still types. Scrolling a pane that has more rows than it can show draws a bar on that pane's right edge. The bar goes away shortly after scrolling stops. A pane whose rows fit has no bar. Shift+Up and Shift+Down scroll the main pane by one output, including a block of several lines.

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
completes memory ids after `@` alongside file paths. The full-screen prompt
does the same: Tab on an `@` token fills one match, extends a shared prefix,
or lists the matches. Text before the `@` stays.

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
on the rewrite. It is off by default; `/refine on` turns it on and persists.
A session that already stored refine-on stays on. When the prompt is changed the REPL
prints `[refine] -> <rewritten prompt>` before any work starts, and the rewrite
is what gets saved to the session. Toggle with `/refine on|off` (persists to
`~/.kdeps/agent-loop-settings.yaml`). Full details: [Prompt refinement](/agent/refine).

## Response rendering

The REPL renders the model's markdown responses - headings, bold, lists, tables, and syntax-highlighted code blocks - in color. It auto-detects the terminal's color depth (truecolor, 256-color, or none) and downsamples the palette to match, so colors render correctly on terminals without 24-bit color (e.g. macOS Terminal.app) instead of collapsing to gray. Output piped to a file is left uncolored.

When extended reasoning is enabled (`/thinking`), the streamed reasoning is rendered as live markdown, updating in place as tokens arrive, shown in muted gray beneath a `* thinking` header and behind a dim left gutter (`|`) so the whole block reads as a distinct aside from the final answer. Inline code renders styled (by color, not literal backticks) in both the reasoning and the response.

## Custom harness

Every piece of text kdeps sends the model to shape its *behavior* - not the conversation itself - is a YAML file too: tool-use rules, the sandbox-hallucination reinforcement, and the system prompts behind compaction, judging, and prompt refinement. Together they are the "harness." Like themes, the built-ins are installed by `kdeps update` (see [Updating assets](/agent/repl-updating#updating-assets)), and you can add your own by dropping a file into `~/.kdeps/harness/<name>.yaml`:

```yaml
# ~/.kdeps/harness/house-style.yaml
name: house-style        # optional - defaults to the filename without its extension
kind: preamble-section    # "preamble-section" (joined into the system prompt) or "standalone" (looked up by name)
order: 200                 # only meaningful for preamble-section; controls where it lands among the others
summary: Answer in one paragraph, then a next-steps list.  # one line sent after the full text has been delivered
body: |-
  Always answer in one paragraph, then a bulleted "next steps" list.
```

A `preamble-section` entry is part of the system prompt. The first request of a session sends every section's `body` in full, and so does the first request after compaction, a fold, or a model switch. Later requests send one `summary` line per section, plus a line telling the model to call the `instruct` tool with that section's name for the full text. `help` is an alias of that tool. `/instruct` stays a slash command. A section with no `summary` falls back to the first sentence of its body. Shipped sections always declare `summary`.

`/efficiency prompt` chooses that later-request behavior and persists it in `~/.kdeps/agent-loop-settings.yaml`:

```text
/efficiency prompt auto      # summary on backends that bill the whole prompt every request (default)
/efficiency prompt full      # always the full sections
/efficiency prompt summary   # summary after the first send, including Anthropic and OpenAI
```

`auto` keeps the full text on Anthropic and OpenAI. Those backends cache a stable system prefix, so a shorter prompt on the next request saves almost nothing and drops the cache. Ollama, llamafile, GGUF, M365, and every other backend are billed for the prompt on every request, so `auto` sends the summaries there.

A file that reuses a built-in's name (e.g. your own `safety.yaml`) replaces that section outright; this includes the safety/accuracy/honesty rules, so overriding one is possible but is your call, the same as a custom theme picking illegible colors.

The built-in `preamble-section` names are `memory`, `tools`, `narration`, `autonomy`, `safety`, `errors`, `scope`, `accuracy`, `honesty`, `code`, `output`, `internals`, `use-kdeps-tools`, and `edit-workflow`. The built-in `standalone` names - looked up individually, not auto-assembled, so a brand-new `standalone` name from you has no effect unless it reuses one of these - are `m365-sandbox`, `tools-reminder`, `skills-preamble`, `compaction-system`, `compaction-user`, `compaction-update-user`, `judge-roster-system`, `judge-system`, `refine-system`, `branch-summary`, and `handshake` (see [Session-integrity handshake](/agent/tools-handshake#session-integrity-handshake)).

`/harness list` shows every section. `/harness enable` and `/harness disable` toggle one and persist it, and the next request sends the full sections again.

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

When a tool stalls (no output for the stall-timeout duration), the default is to auto-increase the timeout by the increment (default 5m) and announce it. Tune both with `/model tool set rounds <n>` and `/model tool set stall-timeout <dur>`. `AutoToolAllocation` (budget) and `AutoStallAllocation` (stall time) are independent and both on by default.

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
