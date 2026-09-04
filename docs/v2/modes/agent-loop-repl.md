# Agent loop REPL features

Runtime behaviors of the interactive agent loop REPL - pasting, rendering, notifications, context size, sessions, and updates. This is **agent mode** only. For starting the loop and registering workflows as tools, see [Agent loop mode](/modes/agent-loop-mode); for the slash commands, see [REPL slash commands](/modes/agent-loop-commands).

## Pasting

Paste a block of text and the REPL treats it as **one prompt**, not one turn per
line - it uses the terminal's bracketed-paste mode, so it works in any modern
terminal, tmux, and screen. What happens next depends on the size:

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

## Response rendering

The REPL renders the model's markdown responses - headings, bold, lists, tables, and syntax-highlighted code blocks - in color. It **auto-detects the terminal's color depth** (truecolor, 256-color, or none) and downsamples the palette to match, so colors render correctly on terminals without 24-bit color (e.g. macOS Terminal.app) instead of collapsing to gray. Output piped to a file is left uncolored.

When extended reasoning is enabled (`/thinking`), the streamed reasoning is rendered as **live markdown**, updating in place as tokens arrive, shown in muted gray beneath a `* thinking` header and behind a dim left gutter (`|`) so the whole block reads as a distinct aside from the final answer. Inline code renders styled (by color, not literal backticks) in both the reasoning and the response.

## Stealth mode

`--stealth` (or `KDEPS_STEALTH=1`, or `/stealth` at runtime) renders the whole REPL - banner, prompt, the text you type, model name, streamed responses, thinking blocks, tool summaries, the `/model` and `/settings` pickers - in near-black dark grays (forced 24-bit color so the shades don't round up on a 256-color terminal). The model name in the status line is the dimmest element on screen, deliberately close to invisible against a dark terminal. Nothing about the output stops working; it just does not read as "an AI session on model X" to anyone glancing at your screen in a cafe, on a plane, or in an open office.

```bash
kdeps --stealth                # start muted
```

```text
/stealth        toggle muted mode
/stealth on     turn it on
/stealth off    turn it off
```

The runtime toggle is remembered - `/stealth on` writes `stealth: true` to `~/.kdeps/agent-loop-settings.yaml`, so the next `kdeps` starts muted too. Precedence: the `--stealth` flag wins, then `KDEPS_STEALTH`, then the persisted setting. The flag and env var override the stored value for that one session without changing it. Stealth affects rendering only - prompts, responses, memory, tool calls, and logs are unchanged.

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

## Sessions

Every conversation is saved as a JSONL file under `~/.kdeps/sessions/`. The session ID is shown at the start of each run. Resume one with:

```bash
kdeps --resume <session-id>
```

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

## See also

- [Agent loop mode](/modes/agent-loop-mode) - starting the loop, tool registration
- [REPL slash commands](/modes/agent-loop-commands) - the full command table
- [Skills and prompt templates](/modes/agent-loop-skills) - context files that teach the agent
- [Local model management](/modes/agent-loop-models) - `/model`, `/context`, running servers
