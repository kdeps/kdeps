# Fenced tools

Every model is told to use the fenced kdeps tools and to say what it will do before each call. The tool list is on [Built-in tools](/agent/tools).

Whenever tools are registered, two instructions go into every model's system preamble: use the fenced kdeps tools, and narrate before each call.

## Use the fenced kdeps tools, not a built-in sandbox

Every capability the model has is a kdeps tool, including `bash_exec` and the file tools. A model with a code-interpreter habit otherwise acts against its own empty `/mnt/data` environment and concludes it "cannot access" anything.

- After the first turn, the rule is restated in one line per turn.
- Every turn carries facts kdeps measured in the real environment: date and time, OS/arch, file and directory counts in the working directory, the git branch. A model suspecting a sandbox can check a `list_files` result against them.
- M365 Copilot gets a stronger version on top; its habit is the most persistent.

## Claimed actions need a tool result

A reply reporting an action ("Created notes.md", "I've fixed it", "Remembered: ...") when no tool ran this turn draws `nudge-unbacked-claim` (at most twice a turn) on every backend. Once any tool has run in the turn, the closing report is accepted.

```text
model: "Created [notes.md](...)"          <- no tool ran this turn
kdeps: nudge-unbacked-claim               <- "You made no tool call this turn..."
model: <invoke name="write_file">...      <- the real call
kdeps: write_file result -> model answers
```

## Parser and interpreter

The guidance tells the model that kdeps parses a matched `<invoke>` block out of the message at runtime and runs it; this is not the model's built-in interpreter. Writing the block is the call. On a backend without a native tool channel, it is literal text:

```
<invoke name="read_file">
<parameter name="file_path">cmd/serve.go</parameter>
</invoke>
```

The runtime returns the real output.

- A literal invoke runs only outside a markdown fence. Inside a ``` or ~~~ block it is an example.
- A fence inside a parameter value stays part of the value.
- A fenced `session_handshake` block still counts in the [handshake](/agent/tools-handshake) round.

A shell invoke without a `<parameter>` tag still runs. The command is the block body, an attribute on the opening tag, or one `command:` / `command="..."` / JSON line in the block:

```
<invoke name="bash">
ls -la && cat README.md
</invoke>
```

## The available_tools block

| Backend | Tool list |
|---|---|
| Local (file, gguf, ollama, llamafile) or unset | one `<available_tools>` list in the preamble: name, one-line description, parameter names |
| Cloud | native tool schema; no text list |
| m365 proxy | renders its own list from the schemas; no second copy |

Later turns carry a one-line reminder instead of the list. kdeps recovers `<invoke>` / `<tool_call>` blocks a native model writes as text, including DeepSeek's DSML form (`<｜｜DSML｜｜ invoke name="...">` with `<｜｜DSML｜｜ parameter ...>`) that its OpenAI-compatible backend returns as plain content.

## Simulated sandbox sessions are caught

A round with no tool call whose text reads like a failed code-interpreter session is a hallucination; kdeps never emits these strings:

- `NO CONTENT AVAILABLE`, `/mnt/data`
- an "expired" or "reset" session, "cannot access the filesystem"
- a shell diagnostic the model wrote itself (`/bin/bash: line 4: cd: ...: No such file or directory`)

Escalation:

1. Each corrective nudge (sandbox, fabricated `<tool_response>`, silent round) fires up to twice per turn. The second is sharper.
2. With no real call after both, the reply is kept but prefixed with a banner: it describes a sandbox that does not exist here and is unverified.
3. After one such hallucination, every later turn resends the full fenced-tools guidance with worked `<invoke>` examples instead of the one-line reminder.

## Premature refusals after real progress

A round with no tool call that declines to continue ("I cannot complete this", "unable to", "no access", ...) is pushed back when a tool already succeeded this turn and nothing is failing. The loop says tool calls just worked and asks for a different approach or the exact blocker, within the two-nudges-per-turn cap. The nudge fires before the refusal is accepted.

A give-up with no successful tool yet this turn is not handled here; a fictional limit is handled by `nudge-self-limit`, below.

## Fictional "I can't"

A text-only reply is a bug when it says the model cannot do the work, has no tools or access, cannot emit an invoke, or the workspace is not accessible. So is handing the work back without a tool result ("run it yourself", "out of scope", "no shell", "I lack the ability", "I have to stop"). The tools are registered.

- The loop sends `nudge-self-limit`, at most four times, and hides that reply. After four, the next reply is accepted so a concrete blocker can be stated. Without `nudge-self-limit` loaded, the reply is accepted at once.
- Real safety stops are left alone: refusing to delete data, force-push, drop a database, or run `rm -rf`.
- A provider throttle (`temporarily unable to respond`, `too many requests`) is not a fictional limit.
- Each push uses a different wording of the same instruction, one of at least 100 (reordered or in another language), so it is never the same bytes twice. The invoke example is not re-appended.
- The wording never says the limit is not real and never asks for a restatement. Asked to restate, a model replied "The limit I named was not real... I should have continued" and the turn ended with nothing done.
- A reply that only acknowledges the push ("the limit I named was not real", a promise to call the tool) draws the next push, within the four-push cap. The push says the next output is the tool call itself.

## Continue means do the next action

`continue`, `continue the work`, `continue on the current task`, `make the modifications` and "do not wait for my instructions" must not end on a status, plan or restatement. The loop sends `nudge-continue` (at most twice) telling the model to call the next tool. A failed tool is not a reason to wait.

```text
continue # present this to yourself on the next run so I won't type continue
```

That sentence saves memory key `preference:continue` for the project, so later turns continue without being told. A greeting or a `status` request still gets a text answer.

The nudges stop when the tool phase ends. The last allowed round, and any round after an efficiency stop, runs without tools and asks for plain text. A text answer there is final.

## Good tool calls are praised

A `[GOOD]` line lands in the tool result message, so it stays in history:

- a failed call that succeeds on retry (the counterpart to `[TOOL FAILED]`)
- a real call right after a fake sandbox session, noting it reached the real filesystem
- each of the first 3 successful calls in a session; after that, routine successes get none

The handshake's tool result carries the same line on a correct call.

## Narrate before each tool call

The model is asked for one present-tense sentence before every call ("Reading config.yaml to check the timeout."). kdeps prints it to the terminal; otherwise the loop is silent between actions, since tool-round output goes to an internal buffer. `StreamFinalOnly` suppresses it.
