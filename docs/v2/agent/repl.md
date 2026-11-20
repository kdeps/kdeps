# Agent loop REPL features

Runtime behaviors of the interactive agent loop REPL: pasting, rendering, notifications, and the tool budget. To start the loop, see the [agent REPL](/agent/). For the slash commands, see [REPL slash commands](/agent/commands).

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

The prompt supports readline editing. kdeps stores history in `~/.kdeps/repl_history`.

| Key | Action |
|---|---|
| `Ctrl+A` / `Ctrl+E` | Start / end of line |
| `Ctrl+B` / `Ctrl+F` | Back / forward one character |
| `Alt+B` / `Alt+F` | Back / forward one word |
| `Ctrl+U` / `Ctrl+K` | Delete before / after the cursor |
| `Ctrl+W`, `Alt+Backspace` | Delete the word before the cursor |
| `Alt+D` | Delete the next word |
| `Ctrl+Y` | Put back the last deleted text |
| `Ctrl+T` | Swap the two characters before the cursor |
| `Ctrl+D` | Delete the character under the cursor; quit on an empty prompt |
| `Ctrl+L` | Clear the screen |
| Up / Down, `Ctrl+P` / `Ctrl+N` | Step through history |

`Ctrl+R` searches history backward: type a substring and the line fills with the newest match. `Ctrl+R` again jumps to the next older match. `Ctrl+S` searches forward. Enter runs the match. `Ctrl+G` cancels.

Some terminals and multiplexers capture `Ctrl+S` for flow control (XON/XOFF). `stty -ixon` in your shell config frees it.

## Multimodal input

Attach images and other binary files to your prompt using `@`.

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
saves the setting. A workflow chat uses `harness.refine: true`.

When refinement changes the prompt, the REPL prints
`[refine] -> <rewritten prompt>` before the work starts. kdeps saves the
rewrite to the session. To change the setting, use `/refine on|off` (saved to
`~/.kdeps/agent-loop-settings.yaml`). For more data, see [Prompt refinement](/agent/refine).

## Planning council

Before a task that is worth planning, a council of members drafts an approach, revises it after they read the other drafts, and ranks the job orders. The turn carries that plan and a progress file. The prompt you typed stays as you typed it. The council is off by default. `/plan on` turns it on and persists to `~/.kdeps/agent-loop-settings.yaml`. A workflow chat uses `harness.plan: true`. Full details: [Planning council](/agent/plan).

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

The default mode is `auto`. Later requests send one summary line per section on backends that do not cache the prompt. Anthropic and OpenAI keep the full text. Set `prompt_mode` to `auto`, `full`, or `summary` in `~/.kdeps/agent-loop-settings.yaml`.

A file with a built-in's name (e.g. your own `safety.yaml`) replaces that section, including the safety, accuracy and honesty rules.

Built-in `preamble-section` names: `memory`, `tools`, `narration`, `autonomy`, `safety`, `errors`, `scope`, `accuracy`, `honesty`, `code`, `output`, `internals`, `use-kdeps-tools`, `edit-workflow`.

Built-in `standalone` names: `tools-reminder`, `skills-preamble`, `compaction-system`, `compaction-user`, `compaction-update-user`, `judge-roster-system`, `judge-system`, `refine-system`, `plan-council-*`, `branch-summary`. kdeps finds standalone entries by name, so a new standalone name has no effect.

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
| Model name, spinner, the context-path status line | [Status line](/agent/repl-status) |
| Token budget, compaction, fold | [Context window](/agent/repl-context) |
| Per-folder history and the resume picker | [Sessions](/agent/repl-sessions) |
| `/upgrade`, nightly, and a pinned version | [Updating kdeps](/agent/repl-updating) |

- [Agent REPL](/agent/) - start the loop
- [REPL slash commands](/agent/commands) - the full command table
- [Skills and prompt templates](/agent/skills) - context files that teach the agent
- [Local model management](/agent/models) - `/model`, `/context`, running servers
