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

The prompt supports readline editing. History is stored in `~/.kdeps/repl_history`; the full-screen prompt shares it.

| Key | Action |
|---|---|
| `Ctrl+A` / `Ctrl+E` | Start / end of line |
| `Ctrl+Left` / `Ctrl+Right` (`Alt+Left` / `Alt+Right`, `Alt+B` / `Alt+F`) | Move by word |
| `Ctrl+U` / `Ctrl+K` | Delete before / after the cursor |
| `Ctrl+W`, `Alt+Backspace` | Delete the word before the cursor |
| `Alt+D` | Delete the next word |
| `Ctrl+D` | Delete the character under the cursor (not saved for `Ctrl+Y`); quit on an empty prompt |
| `Ctrl+Y` | Put back text removed by `Ctrl+K`, `Ctrl+U`, `Ctrl+W`, `Alt+D` or `Ctrl+Shift+X`; repeated removals with no typing between join into one yank |
| Up / Down | Step through history |
| `Ctrl+P` / `Ctrl+N` | Previous / next history line (full-screen) |
| `Ctrl+O` | Edit the line in `$KDEPS_EDITOR`, then `$VISUAL`, `$EDITOR`, `vi` |

Full-screen prompt only:

| Input | Action |
|---|---|
| `Shift+Left` / `Shift+Right`, drag | Select |
| `Ctrl+Shift+A` | Select the whole line |
| `Ctrl+Shift+C` | Copy the selection; with none, copy the visible main pane as one line |
| `Ctrl+Shift+X` | Cut the selection |
| Typing, Backspace | Replace the selection |
| Esc | Clear the selection; on a recalled line, restore the unsent text (does not quit) |
| Click | Move the cursor (prompt) or focus the pane for arrow keys (main pane, feed) |
| Double / triple click | Select the word / line |

- An edit on a recalled line is kept, so the next `Ctrl+N` does not replace it.
- `Ctrl+O`: saving replaces the prompt without sending it. Newlines become spaces, a blank file clears the prompt, and a failed editor leaves the prompt unchanged.
- A prompt longer than the row scrolls to keep the cursor visible. A click lands on the drawn letter.

`Ctrl+R` searches history backward: type a substring and the line fills with the newest match. `Ctrl+R` again jumps to the next older match; `Ctrl+S` searches forward. Enter runs the match; `Ctrl+G` or Esc cancels.

Some terminals and multiplexers capture `Ctrl+S` for flow control (XON/XOFF). `stty -ixon` in your shell config frees it.

## Full-screen scrolling

In a terminal, the agent loop (`kdeps [path]`) draws the session full-screen on the alternate screen. Workflow and agency runs (`kdeps run`) never do. Piped or redirected input or output gets the line REPL. `KDEPS_TUI=0` forces the line REPL; `KDEPS_TUI=1` forces full-screen.

| Key | Action |
|---|---|
| Up / Down | Scroll the focused pane one row |
| `Shift+Up` / `Shift+Down` | Scroll the main pane one output (a multi-line block counts as one) |
| `Ctrl+Up` / `Ctrl+Down` | Scroll half a page |
| PgUp / PgDown | Scroll a page |
| Home / End | Oldest row / newest row (End resumes following) |
| `Ctrl+End` | Follow the main pane, even when the feed is focused |
| `Ctrl+Space` | Fold the bottom output of the main pane to its first line, or unfold it |
| `Ctrl+Shift+V` | Paste clipboard text into the prompt (newlines become spaces) |
| `Ctrl+G` | Open or close the key list |
| `Ctrl+T` | Hide or show the header pills (model name and title stay) |
| `Ctrl+Z` | Suspend to the shell; the view returns when that shell exits |

- Scrolling stops at either end, so one PgDown from the top moves forward.
- On an older row the status row says `not following`, and new output does not move the view. End, scrolling to the newest row, or sending a prompt follows again.
- A pane with more rows than it shows draws a scroll bar on its right edge while scrolling. It fades shortly after.
- One-line outputs do not fold. An empty clipboard leaves the prompt unchanged.
- Plain `Space`, `z`, `v`, `g` and `t` still type.

## Multimodal input

Attach images and other binary files to your prompt using `@`. In the full-screen prompt, Ctrl+V pastes a clipboard image as `@path`. A clipboard path to an image file is pasted the same way. Text and an empty clipboard leave the prompt. `v` still types.

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

With `/thinking` on, streamed reasoning renders as live markdown in muted gray under a `* thinking` header, behind a dim `|` gutter. Inline code renders by color, not literal backticks, in reasoning and response.

## Custom harness

The harness is every text kdeps sends to shape the model's behavior: tool-use rules, the sandbox-hallucination reinforcement, and the compaction, judge and prompt-refinement system prompts. Each is a YAML file. `kdeps update` installs the built-ins (see [Updating assets](/agent/repl-updating#updating-assets)); add your own as `~/.kdeps/harness/<name>.yaml`:

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

A file with a built-in's name (e.g. your own `safety.yaml`) replaces that section, including the safety, accuracy and honesty rules.

Built-in `preamble-section` names: `memory`, `tools`, `narration`, `autonomy`, `safety`, `errors`, `scope`, `accuracy`, `honesty`, `code`, `output`, `internals`, `use-kdeps-tools`, `edit-workflow`.

Built-in `standalone` names: `m365-sandbox`, `tools-reminder`, `skills-preamble`, `compaction-system`, `compaction-user`, `compaction-update-user`, `judge-roster-system`, `judge-system`, `refine-system`, `branch-summary`, `handshake` (see [Session-integrity handshake](/agent/tools-handshake#session-integrity-handshake)). Standalone entries are looked up by name, so a new standalone name has no effect.

`/harness list` shows every section. `/harness enable` and `/harness disable` toggle one and persist it, and the next request sends the full sections again.

## Turn-complete alert

When a turn takes a while, the REPL rings the terminal and posts a desktop notification when the response is ready:

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
