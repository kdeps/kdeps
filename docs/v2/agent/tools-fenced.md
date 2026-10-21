# Fenced tools

Every model is told to call the fenced kdeps tools, and to say what it is about to do before the call. The tool list stays on [Built-in tools](/agent/tools).

## Fenced tools only, and narration

Two instructions go into the system preamble for every model whenever tools are registered:

## Use the fenced kdeps tools, not a built-in sandbox

Every capability the model has here is a kdeps tool - including `bash_exec` and the file tools. A model trained with a code-interpreter / "run code" / "analysis" habit will otherwise act against its own empty `/mnt/data`-style environment and conclude it "cannot access" anything. The rule is restated in one line on every turn after the first. Every turn also carries a line of facts kdeps measured itself in the real environment - date and time, OS/arch, how many files and directories are in the working directory, the git branch - so a model that suspects a sandbox can check a `list_files` result against them. M365 Copilot gets an extra, stronger version of this on top (its habit is the most persistent).

## Parser and interpreter

The guidance also tells the model that kdeps is a parser and an interpreter: it parses a matched `<invoke>` block out of the message at runtime and interprets it. That is not the model's built-in code interpreter. Writing the block is the call, not an example. On a backend with no native tool channel, that block is a LITERAL invoke block: the characters `<invoke>` and `</invoke>` written as text in the message. One matched block does it -

```
<invoke name="read_file">
<parameter name="file_path">cmd/serve.go</parameter>
</invoke>
```

and the runtime hands back the real output. A literal invoke runs only when those tags sit outside a markdown fence. The same tags inside a ``` or ~~~ block are an example and are not run. A fence inside a parameter value stays part of that value. A fenced `session_handshake` block still counts in the handshake round.

A shell invoke with no `<parameter>` tag still runs. The command is the block body, an attribute on the opening tag, or one `command:` / `command="..."` / JSON line inside the block:

```
<invoke name="bash">
ls -la && cat README.md
</invoke>
```

## The available_tools block

The same preamble lists every registered tool in an `<available_tools>` block: name, parameters, and an `<invoke>` skeleton for each. That block is what the model copies a name from. Later turns repeat the list next to the one-line reminder. Backends with a native tool channel (Anthropic, OpenAI) also receive the tools as a native schema; kdeps recovers an `<invoke>` / `<tool_call>` block written as text if a native model emits one anyway, including DeepSeek's DSML-wrapped form (`<｜｜DSML｜｜ invoke name="...">` with `<｜｜DSML｜｜ parameter ...>`), which its OpenAI-compatible backend emits as plain content.

## Simulated sandbox sessions are caught, and escalated

When a round makes no tool call but the text reads like a failed code-interpreter session - `NO CONTENT AVAILABLE`, `/mnt/data`, an "expired" or "reset" session, "cannot access the filesystem", or a shell diagnostic the model wrote itself (`/bin/bash: line 4: cd: ...: No such file or directory`) - the loop treats it as a hallucination (kdeps never emits those strings and nothing ran). A model that repeats the same hallucination a second time within one turn gets a second, sharper nudge instead of the turn silently accepting the repeat as a real answer - each corrective nudge (sandbox, fabricated `<tool_response>`, silent round) fires up to twice per turn, not once. If the model still has not made a real call after both nudges, the turn does not settle on the fake transcript unflagged: in goal mode the active task is recorded failed, not done; otherwise the returned text is prefixed with a clear "this describes a sandbox that doesn't exist here, treat it as unverified" banner (the model's words are kept, just flagged). Separately, once a session has produced even one such hallucination, every later turn resends the full fenced-tools guidance (with the worked `<invoke>` examples) instead of the one-line reminder - a model that has shown this failure mode gets more reinforcement, not less.

## A premature "sorry, I can't" is pushed back on when there was real progress

When a round ends with no tool call and the text reads as declining to continue ("I cannot complete this," "I couldn't complete this," "unable to," "no access," ...), but a tool call already succeeded earlier this turn and nothing is currently failing, the loop does not accept the refusal as final - it tells the model tool calls just worked and to try a different approach, or state exactly what is blocking it, bounded to the same two-nudges-per-turn cap as every other corrective nudge here. A give-up reply with no successful tool yet this turn is not handled by that progress nudge. A fictional limit in that reply is handled by `nudge-self-limit`, below. This applies with or without goal mode: the nudge fires before the active task would be recorded failed, so a premature refusal is not logged as a settled failure while the nudge budget remains.

## A fictional "I can't" is pushed back on

A text-only reply that says the model cannot do the work, has no tools, has no access, cannot emit an invoke, or claims the workspace is not accessible in this environment is a bug. So is a paraphrase that hands the work back ("run it yourself", "out of scope", "no shell", "I lack the ability", "I have to stop") without a tool result. The tools are already registered. The loop sends `nudge-self-limit` (at most four times) and does not show that reply. A real safety stop is left alone: refusing to delete data, force-push, drop a database, or run `rm -rf`. After four pushes, the next reply is accepted so a concrete blocker can be stated. If `nudge-self-limit` is not loaded, that reply is accepted immediately. The loop does not keep appending the invoke example. Each push adds a different wording of that same instruction, one of at least 100 (the same order in another sentence or another language), so the request is not the same bytes as the push before it. The wording never states that the limit is not real and never asks for a restatement: a model answered "say it again, in new words: the limit you named is not real" with "The limit I named was not real. The tool call succeeded, and I should have continued..." and the turn ended there with nothing done. A reply that only acknowledges the push-back - "the limit I named was not real", a promise to call the tool - draws the next push instead of being accepted, within the same four-push cap, and the push tells the model its next output is the tool call itself, not a reply. A provider throttle (`temporarily unable to respond`, `too many requests`) is not a fictional limit.

## Continue means do the next action

`continue`, `continue the work`, `continue on the current task`, `make the modifications`, and "do not wait for my instructions" do not end on a status, a plan, or a restatement. The loop sends `nudge-continue` (at most twice) telling the model to call the next tool. A failed tool is not a reason to wait.

```text
continue # present this to yourself on the next run so I won't type continue
```

That sentence also saves memory key `preference:continue` for the project. Later turns follow it without another `continue`. A greeting or a `status` request still answers in text.

These nudges stop once the tool phase ends. The last allowed round, and any round after an efficiency stop, runs with the tools stripped and a prompt telling the model to answer in plain text without tool-call markup. A text answer there is the turn's final answer; pushing it to call a tool would contradict that directive and no call could run anyway.

## Good tool-calling behavior is praised, not just bad behavior flagged

Three cases, each a `[GOOD]` acknowledgment landing in the tool result message itself - part of conversation history, not just the terminal, so it carries forward into later tasks in the session:

- A failed call that then succeeds on retry - the positive counterpart to `[TOOL FAILED]`.
- A real tool call right after a fake sandbox session was detected - specifically calling out that this one reached the real filesystem, not the simulated one.
- Any of the first 3 successful tool calls in a session, even with no prior failure - building the habit early. After that, an ordinary success gets no banner; constant praise for routine calls would just be noise.

The mandatory handshake's own tool result carries the same kind of line on a correct call, and a retry nudge there reshows the exact `<invoke>` syntax with a running miss count, framed as help rather than a rebuke.

## Narrate before each tool call

The model is asked to say in one present-tense sentence what it is about to do ("Reading config.yaml to check the timeout.") before every call. That line is printed to the terminal - without it the loop is silent between actions, because the streamer writes tool-round output to an internal buffer. Suppressed when `StreamFinalOnly` is set.
