# Fenced tools

Every model is told to call the fenced kdeps tools, and to say what it is about to do before the call. The tool list stays on [Built-in tools](/agent/tools).

## Fenced tools only, and narration

Two instructions go into the system preamble for **every** model whenever tools are registered:

## Use the fenced kdeps tools, not a built-in sandbox

Every capability the model has here is a kdeps tool - including `bash_exec` and the file tools. A model trained with a code-interpreter / "run code" / "analysis" habit will otherwise act against its own empty `/mnt/data`-style environment and conclude it "cannot access" anything. The rule is restated in one line on every turn after the first. Every turn also carries a line of facts kdeps measured itself in the real environment - date and time, OS/arch, how many files and directories are in the working directory, the git branch - so a model that suspects a sandbox can check a `list_files` result against them. M365 Copilot gets an extra, stronger version of this on top (its habit is the most persistent).

## Parser and interpreter

The guidance also tells the model that kdeps is a parser and an interpreter: it parses a matched `<invoke>` block out of the message at runtime and interprets it. That is not the model's built-in code interpreter. Writing the block is the call, not an example. On a backend with no native tool channel, that block is a LITERAL invoke block: the characters `<invoke>` and `</invoke>` written as text in the message. One matched block does it -

```
<invoke name="read_file">
<parameter name="file_path">cmd/serve.go</parameter>
</invoke>
```

and the runtime hands back the real output.

## The available_tools block

The same preamble lists every registered tool in an `<available_tools>` block: name, parameters, and an `<invoke>` skeleton for each. That block is what the model copies a name from. Later turns repeat the list next to the one-line reminder. Backends with a native tool channel (Anthropic, OpenAI) also receive the tools as a native schema; kdeps recovers an `<invoke>` / `<tool_call>` block written as text if a native model emits one anyway.

## Simulated sandbox sessions are caught, and escalated

When a round makes no tool call but the text reads like a failed code-interpreter session - `NO CONTENT AVAILABLE`, `/mnt/data`, an "expired" or "reset" session, "cannot access the filesystem" - the loop treats it as a hallucination (kdeps never emits those strings and nothing ran). A model that repeats the same hallucination a second time within one turn gets a second, sharper nudge instead of the turn silently accepting the repeat as a real answer - each corrective nudge (sandbox, fabricated `<tool_response>`, silent round) fires up to twice per turn, not once. If the model still hasn't made a real call after both nudges, the turn does not settle on the fake transcript unflagged: in goal mode the active task is recorded failed, not done; otherwise the returned text is prefixed with a clear "this describes a sandbox that doesn't exist here, treat it as unverified" banner (the model's words are kept, just flagged). Separately, once a session has produced even one such hallucination, every later turn resends the full fenced-tools guidance (with the worked `<invoke>` examples) instead of the one-line reminder - a model that has shown this failure mode gets more reinforcement, not less.

## A premature "sorry, I can't" is pushed back on when there was real progress

When a round ends with no tool call and the text reads as declining to continue ("I cannot complete this," "unable to," "no access," ...), but a tool call already succeeded earlier this turn and nothing is currently failing, the loop does not accept the refusal as final - it tells the model tool calls just worked and to try a different approach, or state exactly what is blocking it, bounded to the same two-nudges-per-turn cap as every other corrective nudge here. A give-up reply with no progress behind it (nothing succeeded yet this turn) is never second-guessed this way - there is nothing to push back with. This applies with or without goal mode: in goal mode it fires *before* the active task would otherwise be recorded failed, so a premature refusal never gets logged as a settled failure while the nudge budget remains.

## Good tool-calling behavior is praised, not just bad behavior flagged

Three cases, each a `[GOOD]` acknowledgment landing in the tool result message itself - part of conversation history, not just the terminal, so it carries forward into later tasks in the session:

- A failed call that then succeeds on retry - the positive counterpart to `[TOOL FAILED]`.
- A real tool call right after a fake sandbox session was detected - specifically calling out that this one reached the real filesystem, not the simulated one.
- Any of the first 3 successful tool calls in a session, even with no prior failure - building the habit early. After that, an ordinary success gets no banner; constant praise for routine calls would just be noise.

The mandatory handshake's own tool result carries the same kind of line on a correct call, and a retry nudge there reshows the exact `<invoke>` syntax with a running miss count, framed as help rather than a rebuke.

## Narrate before each tool call

The model is asked to say in one present-tense sentence what it is about to do ("Reading config.yaml to check the timeout.") before every call. That line is printed to the terminal - without it the loop is silent between actions, because the streamer writes tool-round output to an internal buffer. Suppressed when `StreamFinalOnly` is set.
