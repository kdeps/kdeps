# Fenced tools

Each model gets an instruction to use the fenced kdeps tools and to say what it will do before each call. The tool list is on [Built-in tools](/agent/tools).

When tools are registered, two instructions go into the system preamble of each model: use the fenced kdeps tools, and narrate before each call.

## Use the fenced kdeps tools, not a built-in sandbox

Every capability the model has is a kdeps tool, including `bash_exec` and the file tools. A model with a code-interpreter habit otherwise acts against its own empty `/mnt/data` environment and concludes it "cannot access" anything.

- After the first turn, kdeps gives the rule again in one line on each turn.
- Every turn carries facts kdeps measured in the real environment: date and time, OS/arch, file and directory counts in the working directory, the git branch. A model suspecting a sandbox can check a `list_files` result against them.
- M365 Copilot gets a stronger version on top. Its habit is the most persistent.

## Claimed actions need a tool result

A reply can report an action ("Created notes.md", "I've fixed it", "Remembered: ...") when no tool ran in this turn. On all backends, this reply causes `nudge-unbacked-claim` (a maximum of two times in a turn). After a tool runs in the turn, kdeps accepts the last report.

```text
model: "Created [notes.md](...)"          <- no tool ran this turn
kdeps: nudge-unbacked-claim               <- "You made no tool call this turn..."
model: <invoke name="write_file">...      <- the real call
kdeps: write_file result -> model answers
```

## Parser and interpreter

The guidance tells the model that kdeps parses a matched `<invoke>` block in the message at runtime and runs it. This is not the built-in interpreter of the model. The block that the model writes is the call. On a backend without a native tool channel, it is literal text:

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

Later turns have a one-line reminder, not the list. A native model can write `<invoke>` / `<tool_call>` blocks as text. kdeps recovers these blocks. This includes the DSML form of DeepSeek (`<｜｜DSML｜｜ invoke name="...">` with `<｜｜DSML｜｜ parameter ...>`), which its OpenAI-compatible backend returns as plain content.

## Simulated sandbox sessions are caught

A round with no tool call whose text reads like a failed code-interpreter session is a hallucination. kdeps never emits these strings:

- `NO CONTENT AVAILABLE`, `/mnt/data`
- an "expired" or "reset" session, "cannot access the filesystem"
- a shell diagnostic the model wrote itself (`/bin/bash: line 4: cd: ...: No such file or directory`)

Escalation:

1. Each corrective nudge (sandbox, fabricated `<tool_response>`, silent round) fires up to twice per turn. The second is sharper.
2. If there is no real call after the two nudges, kdeps keeps the reply but puts a banner before it. The banner says that the reply describes a sandbox that does not exist here, and that it is not verified.
3. After one hallucination of this type, each later turn sends the full fenced-tools guidance again with complete `<invoke>` examples. It does not use the one-line reminder.

## Premature refusals after real progress

A round with no tool call can refuse to continue ("I cannot complete this", "unable to", "no access", ...). If a tool was successful in this turn and nothing fails, kdeps does not accept this refusal. The loop says that tool calls were successful. It asks for a different method or the exact blocker, with a maximum of two nudges in a turn. The nudge occurs before kdeps accepts the refusal.

This rule does not apply if no tool was successful in this turn. `nudge-self-limit` (below) controls a fictional limit.

## Fictional "I can't"

A text-only reply is a bug in these conditions: it says that the model cannot do the work, has no tools or access, or cannot write an invoke. It is also a bug if it says that the workspace is not accessible. A reply that gives the work back to you without a tool result is also a bug ("run it yourself", "out of scope", "no shell", "I lack the ability", "I have to stop"). The tools are registered.

- The loop sends `nudge-self-limit`, a maximum of four times, and hides that reply. After four times, kdeps accepts the next reply, so that the model can give a specific blocker. If `nudge-self-limit` is not loaded, kdeps accepts the reply immediately.
- kdeps does not change real safety stops: a refusal to delete data, force-push, drop a database, or run `rm -rf`.
- A provider throttle (`temporarily unable to respond`, `too many requests`) is not a fictional limit.
- Each push uses a different wording of the same instruction. There are 100 or more wordings (with a different order, or in a different language). For this reason, the bytes are never the same two times. kdeps does not add the invoke example again.
- The wording never says that the limit is not real, and never asks the model to say it again. In one test, a model replied "The limit I named was not real... I should have continued" and the turn ended with no work done.
- A reply can only acknowledge the push ("the limit I named was not real", or a promise to call the tool). This reply causes the next push, with a maximum of four pushes. The push says that the next output must be the tool call.

## Continue means do the next action

`continue`, `continue the work`, `continue on the current task`, `make the modifications` and "do not wait for my instructions" must not end on a status, plan or restatement. The loop sends `nudge-continue` (at most twice) telling the model to call the next tool. A failed tool is not a reason to wait.

```text
continue # present this to yourself on the next run so I won't type continue
```

That sentence saves memory key `preference:continue` for the project, so later turns continue without an instruction. A greeting or a `status` request gets a text answer.

The nudges stop when the tool phase ends. The last permitted round, and all rounds after an efficiency stop, run without tools and ask for plain text. A text answer there is final.

## Good tool calls are praised

A `[GOOD]` line lands in the tool result message, so it stays in history:

- a failed call that succeeds on retry (the counterpart to `[TOOL FAILED]`)
- a real call right after a fake sandbox session, noting it reached the real filesystem
- each of the first 3 successful calls in a session. After that, routine successes get none

The handshake's tool result carries the same line on a correct call.

## Narrate before each tool call

kdeps asks the model for one present-tense sentence before each call ("Reading config.yaml to check the timeout."). kdeps prints it to the terminal. Without it, the loop is silent between actions, because tool-round output goes to an internal buffer. `StreamFinalOnly` suppresses it.
