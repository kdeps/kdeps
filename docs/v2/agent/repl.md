# Agent loop REPL features

Runtime behaviors of the interactive agent loop REPL - pasting, rendering, notifications, and the tool budget. This is agent mode only. For starting the loop and registering workflows as tools, see [Agent mode](/agent/). For the slash commands, see [REPL slash commands](/agent/commands).

## Pasting

When you paste a block of text, the REPL uses it as one prompt, not as one turn
for each line.

The REPL uses the bracketed-paste mode of the terminal. It
enables this mode again before each prompt. A child process (an editor that
you open with `!`, or a pager) or a terminal reset can disable it. With the
mode enabled again, a later paste does not split into one submission for each
line. This operates in all modern terminals, tmux, and screen. The next step depends on the size:

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

The REPL inserts a **small** single-line paste as usual editable text on the input line.

The REPL saves all other pastes to memory with an id like `paste:3f9a1c2e`. The
memory is the folder memory: a note in the `memory/` folder of the project in
the knowledge vault. A multi-line paste with a maximum of 4 lines, 20 words, and
240 characters shows exactly on the edit line.

A larger paste shows as
`@paste:3f9a1c2e`, so that the prompt and your scrollback stay easy to read.
Press Enter one time to submit. The reference expands to the pasted text, so
the model always gets the full paste.

```text
you> why does this fail: @paste:3f9a1c2e      <- what the line and history keep
model sees: why does this fail:
--- @paste:3f9a1c2e ---
<the full pasted text>
```

REPL history (Up arrow, `Ctrl+R`) stores the `@paste:<id>` reference, not the
text. Recalling it in a later session still works, because the paste lives in
the folder's memory.

The paste is one character on the edit line, so you can **edit around it**. To
move before or after it, use the arrow keys (or `Ctrl+A` / `Ctrl+E`). Then type
there. For example, paste a stack trace, type `why does this happen: ` in front
of it, and submit.

## Line editing and history search

The prompt supports readline editing. kdeps stores history in `~/.kdeps/repl_history`. The full-screen prompt uses the same history.

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
| Drag the feed border | Resize the feed; the main pane keeps at least 20 columns, the feed at least 30 |

- kdeps keeps an edit on a recalled line, so the next `Ctrl+N` does not replace it.
- `Ctrl+O`: when you save, the text replaces the prompt. kdeps does not send it. Newlines become spaces, a blank file clears the prompt, and a failed editor leaves the prompt unchanged.
- A prompt longer than the row scrolls to keep the cursor visible. A click lands on the drawn letter.

`Ctrl+R` searches history backward: type a substring and the line fills with the newest match. `Ctrl+R` again jumps to the next older match. `Ctrl+S` searches forward. Enter runs the match. `Ctrl+G` or Esc cancels.

Some terminals and multiplexers capture `Ctrl+S` for flow control (XON/XOFF). `stty -ixon` in your shell config frees it.

## Full-screen scrolling

In a terminal, the agent loop (`kdeps [path]`) draws the session full-screen on the alternate screen. Workflow and agency runs (`kdeps run`) never do. Piped or redirected input or output gets the line REPL. `KDEPS_TUI=0` forces the line REPL. `KDEPS_TUI=1` forces full-screen.

The view stays live during a turn. If you send a prompt before the turn ends, the prompt waits and runs next, in order. `/model` opens its picker immediately. The model that you select starts after the turn.

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
- If a pane has more rows than it can show, it draws a scroll bar on its right edge when you scroll. The scroll bar fades a short time after.
- One-line outputs do not fold. An empty clipboard leaves the prompt unchanged.
- Plain `Space`, `z`, `v`, `g` and `t` still type.

## Multimodal input

Attach images and other binary files to your prompt using `@`. In the full-screen prompt, Ctrl+V pastes a clipboard image as `@path`. Ctrl+V pastes a clipboard path to an image file in the same way. Text and an empty clipboard do not change the prompt. `v` types as usual.

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

- kdeps sends image and binary refs (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`, `.tiff`, `.pdf`, `.mp3`, `.mp4`, `.wav`) as multimodal content to the LLM.
- kdeps expands text file refs inline in the prompt.
- kdeps does not change refs that it cannot resolve (file not found, access denied).

## Memory references

`@<memory-id>` expands to that memory entry's value, the same way `@notes.txt`
expands to the file's contents. Any memory id works: pastes (`paste:<id>`),
entries that the model saved with `memory_save`, or ids that `/memory` lists.
After `@`, Tab completes memory ids and file paths.

The full-screen prompt
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

A file that exists has priority over a memory id with the same name. kdeps does
not change an unknown id in the text.

## Prompt refinement

Before each turn, kdeps can run one low-cost LLM call. This call rewrites a
short or incomplete prompt into a clearer, self-contained version. The turn
then runs on the rewrite. It is off by default. `/refine on` turns it on and
saves the setting. A session that already stored refine-on stays on.

When refinement changes the prompt, the REPL prints
`[refine] -> <rewritten prompt>` before the work starts. kdeps saves the
rewrite to the session. To change the setting, use `/refine on|off` (saved to
`~/.kdeps/agent-loop-settings.yaml`). For more data, see [Prompt refinement](/agent/refine).

## Planning council

Before a task that is worth planning, a council of members drafts an approach, revises it after they read the other drafts, and ranks the job orders. The turn carries that plan and a progress file. The prompt you typed stays as you typed it. The council is on unless `/plan off` (persists to `~/.kdeps/agent-loop-settings.yaml`). Full details: [Planning council](/agent/plan).

## Response rendering

The REPL renders the model's markdown responses - headings, bold, lists, tables, and syntax-highlighted code blocks - in color. It finds the color depth of the terminal automatically (truecolor, 256-color, or none) and reduces the palette to match. As a result, colors show correctly on terminals without 24-bit color (for example, macOS Terminal.app), and do not become gray. Output that goes to a file has no color.

With `/thinking` on, streamed reasoning renders as live markdown in muted gray under a `* thinking` header, behind a dim `|` gutter. Inline code renders by color, not literal backticks, in reasoning and response.

## Custom harness

The harness is every text kdeps sends to shape the model's behavior: tool-use rules, the sandbox-hallucination reinforcement, and the compaction, judge and prompt-refinement system prompts. Each is a YAML file. `kdeps update` installs the built-ins (see [Updating assets](/agent/repl-updating#updating-assets)). Add your own as `~/.kdeps/harness/<name>.yaml`:

```yaml
# ~/.kdeps/harness/house-style.yaml
name: house-style        # optional - defaults to the filename without its extension
kind: preamble-section    # "preamble-section" (joined into the system prompt) or "standalone" (looked up by name)
order: 200                 # only meaningful for preamble-section; controls where it lands among the others
summary: Answer in one paragraph, then a next-steps list.  # one line sent after the full text has been delivered
body: |-
  Always answer in one paragraph, then a bulleted "next steps" list.
```

A `preamble-section` entry is part of the system prompt. The first request of a session sends the full `body` of each section. The first request after compaction, a fold, or a model switch also sends the full body. Later requests send one `summary` line for each section. They also send a line that tells the model how to get the full text: call the `instruct` tool with the name of the section.

`help` is an alias of that tool. `/instruct` stays a slash command. A section with no `summary` falls back to the first sentence of its body. Shipped sections always declare `summary`.

`/efficiency prompt` chooses that later-request behavior and persists it in `~/.kdeps/agent-loop-settings.yaml`:

```text
/efficiency prompt auto      # summary on backends that bill the whole prompt every request (default)
/efficiency prompt full      # always the full sections
/efficiency prompt summary   # summary after the first send, including Anthropic and OpenAI
```

`auto` keeps the full text on Anthropic and OpenAI. Those backends cache a stable system prefix, so a shorter prompt on the next request saves almost nothing and drops the cache. Ollama, llamafile, GGUF, M365, and all other backends charge for the prompt on each request, so `auto` sends the summaries there.

A file with a built-in's name (e.g. your own `safety.yaml`) replaces that section, including the safety, accuracy and honesty rules.

Built-in `preamble-section` names: `memory`, `tools`, `narration`, `autonomy`, `safety`, `errors`, `scope`, `accuracy`, `honesty`, `code`, `output`, `internals`, `use-kdeps-tools`, `edit-workflow`.

Built-in `standalone` names: `m365-sandbox`, `tools-reminder`, `skills-preamble`, `compaction-system`, `compaction-user`, `compaction-update-user`, `judge-roster-system`, `judge-system`, `refine-system`, `branch-summary`, `handshake` (see [Session-integrity handshake](/agent/tools-handshake#session-integrity-handshake)). kdeps finds standalone entries by name, so a new standalone name has no effect.

`/harness list` shows every section. `/harness enable` and `/harness disable` toggle one and persist it, and the next request sends the full sections again.

## Turn-complete alert

When a turn takes a long time, the REPL rings the terminal and shows a desktop notification when the response is ready:

- The terminal bell marks the tab/window as having activity in most terminals, tmux, and screen.
- An **OSC 9** desktop notification (`kdeps: response ready`) shows in terminals that support it (iTerm2, WezTerm, kitty). Other terminals ignore it silently.

Only turns longer than a threshold alert, so quick replies stay quiet.

| Env var | Effect |
|---------|--------|
| `KDEPS_NOTIFY=off` | Disable the alert entirely |
| `KDEPS_NOTIFY_MIN=<dur>` | Minimum turn duration to alert (default `10s`; `0` = every turn) |

## Auto-retry

kdeps automatically tries temporary LLM errors (HTTP 429, 5xx, network timeouts) again, a maximum of 3 times, with exponential backoff (2s, 4s, 8s). It does not try context-overflow and authentication errors again.

## Tool budget and stall timeout

The agent loop tracks a tool budget (`MaxToolRounds`) that limits how many tool calls the agent can make per turn. When the budget is almost empty, the REPL shows interactive options: `(i)ncrease` the budget (adds 100 rounds), `(c)hange` to a specific number (`0` = unlimited), or `(i)gnore` to continue. When you enable `AutoToolAllocation` in the configuration, the budget increases automatically without a prompt.

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
