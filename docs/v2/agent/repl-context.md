# Context window

The context window is the token budget for one turn: system prompt, memory, history, and the reply. Compaction and fold shrink a session that no longer fits. The other REPL behaviors stay on [REPL features](/agent/repl).

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

## Text-embedded tool calls

Most models call tools through the backend's tool-use channel. Some instead
write the call as text --- `<tool_call>{"name":...,"arguments":{...}}</tool_call>`,
`<function=name>{...}</function>`, `<invoke name="..."><parameter name="...">...</parameter></invoke>`,
or a bare JSON object --- and sometimes follow it with a **self-written
`<tool_response>`** block and a false "done".

kdeps is a parser and an interpreter. It parses a text-written tool call out of the message at runtime, including a matched `<invoke>` block, and interprets it for real. That is not the model's code interpreter. Without the native tool channel, that block is a LITERAL invoke block: the tags written as text. The system preamble lists every registered tool in `<available_tools>`, each with an `<invoke>` skeleton, and later turns repeat that list. A model-authored
`<tool_response>` is always a hallucination (only the runtime produces tool
results): kdeps strips it, does not accept the turn as finished, and nudges the
model once to make the actual call and wait for the real result. These markers
never reach the visible answer.

## In-turn tool history

A single turn can make many tool calls (`/model tool set rounds <n>`, default 200). The transcript of those calls and their results is re-sent to the model on every round, so a long tool loop can otherwise grow the request without bound - `/compact` and the auto-compaction that runs between turns do not fire mid-turn.

kdeps caps the in-flight transcript at roughly 24K tokens. Past that, the oldest complete tool round-trips are dropped (the leading conversation and this turn's question are always kept), and long tool-call arguments on the older kept steps are replaced with a `[N chars omitted]` placeholder. When this happens you see:

```
[context] trimmed 3 old tool step(s) to fit the window
```

The model keeps the most recent steps in full and a note that earlier ones were dropped. Tool *results* are separately capped at 16 KB each before they ever enter the transcript.

## Compacting the saved conversation

Only the user prompt and the final answer of each turn are saved to the session
- a turn's own tool transcript is never persisted. `/compact` summarizes the
older saved turns and keeps the last few verbatim; it runs whenever the session
has more than a handful of turns, regardless of size. Auto-compaction between
turns fires only once the saved conversation exceeds the token budget (`/model
tool set compact-threshold <n>`), which for a long run of short-but-tool-heavy
turns may never happen - use `/compact` directly. Neither affects a running
turn's tool output; that is bounded by the in-flight window above.

A heading the summarizer leaves blank is dropped before the summary is saved
or shown. That includes a bare `Summary` title and a `## Goal` with nothing
under it. The auto-compact line prints the first sentence of what remains,
not the heading.

The summary from that compact is not itself a reason to compact again. The
next one waits until real conversation, not the summary, has fallen outside
the kept window.

Both the compact budget (how much recent conversation stays verbatim) and the
auto-compaction threshold default to 3/4 of the model's known context window,
not a flat token count - a session on a small local model and one on a
200K-token cloud model each get a sensible default without you having to set
`/model tool set compact-budget <n>` yourself. Switching models with `/model`
recomputes both immediately for the new model; an unrecognized model keeps a
conservative flat default.

The conversation text being summarized is never re-evaluated as a kdeps
expression or template, even if it contains double-curly-brace syntax (kdeps
expressions you were discussing, another template language's tags, or
anything else that looks like one) - it reaches the summarizer exactly as
written.

## Fold

`/fold` is a lighter, more frequent cousin of `/compact` - instead of waiting
for the full compact-threshold safety net, it checkpoints the conversation
(same structured Goal/Progress/Decisions summary, saved to
[memory](/agent/memory-internals#checkpoint-summaries)) every time a
configurable amount of *new* conversation has accumulated since the last
checkpoint. It is on by default (2000 tokens, keeping the last 5 checkpoints
in the prompt's memory block) and needs no setup.

| Command | Effect |
|---|---|
| `/fold` | Show status: auto on/off, threshold, items cap, tokens accumulated since the last checkpoint |
| `/fold now` | Force a fold immediately, regardless of the threshold or the compaction budget |
| `/fold auto` / `/fold auto off` | Toggle automatic folding (default: on) |
| `/fold threshold <n>` | Token delta since the last checkpoint that triggers a fold (e.g. `4k`) |
| `/fold items <n>` | How many recent checkpoints stay in the active memory-prompt window |
| `/fold preset tight\|balanced\|loose` | Named threshold+items bundles - `balanced` is the shipped default |

All four settings persist across sessions, the same way `/model tool set
compact-threshold`/`compact-budget` already do.

`/fold now` is a *forced* fold: it ignores the threshold and the compaction
budget entirely and always summarizes everything except the last few turns,
same as `/compact`. The only way it comes back with nothing is too few turns
to have anything beyond what it always keeps verbatim - it then prints the
turn count needed and the current `sent`/`generated` token counter, a number to
check instead of just an assertion.

Both `/compact` and `/fold` reset `sent` and `generated` to the context that
remains. `sent` is that context (system prompt, memory, and the kept
history). `generated` is the model-written part still in it, including the
new summary. The old cumulative totals are dropped.
