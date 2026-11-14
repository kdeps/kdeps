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

Most models call tools through the tool-use channel of the backend. Some models
write the call as text. Examples are `<tool_call>{"name":...,"arguments":{...}}</tool_call>`,
`<function=name>{...}</function>`, `<invoke name="..."><parameter name="...">...</parameter></invoke>`,
or a bare JSON object. Sometimes, after the call, the model writes its **own
`<tool_response>`** block and a false "done".

kdeps is a parser and an interpreter. At runtime, it parses a tool call that is written as text in the message, also a matched `<invoke>` block. Then it runs the call. This is not the code interpreter of the model. Without the native tool channel, that block is a LITERAL invoke block: the tags are written as text.

On a local backend (file, gguf, ollama, llamafile), the system preamble lists all registered tools in a short `<available_tools>` block. The block gives the name, a one-line description, and the parameter names. Cloud backends send native schemas and do not send this text list. Later turns give a one-line reminder and do not repeat the list.

A `<tool_response>` that the model writes is always a hallucination, because
only the runtime makes tool results. kdeps removes it and does not accept that
the turn is complete. It tells the model one time to make the real call and
wait for the real result. These markers never show in the visible answer.

## In-turn tool history

A single turn can make many tool calls (`/model tool set rounds <n>`, default: the `agent-max-rounds` event, 200). kdeps sends their transcript again on each round. `/compact` and auto-compaction do not start during a turn. For this reason, without a limit, a long tool loop makes the request larger and larger.

kdeps sets a limit of about 12K tokens on the in-flight transcript. After that limit, kdeps removes the oldest complete tool round-trips. The first part of the conversation and the question of this turn stay. On older steps that stay, long arguments become a `[N chars omitted]` placeholder. You see:

```
[context] trimmed 3 old tool step(s) to fit the window
```

The model keeps the most recent steps in full, and a note that kdeps removed earlier steps. Before a tool result goes into the transcript, kdeps cuts it to a maximum of 8 KB. It keeps the start and the end of the result, where test totals and errors usually are. kdeps cuts results older than the two most recent steps to 1 KB, with a note. If the model needs the remaining part, it runs the tool again.

## Tool definitions on demand

Every tool definition is re-sent on every round, so 70+ tools cost about 12K tokens a round. kdeps sends the full definition of a core set only (`bash_exec`, the file tools, `search_local`, the memory tools, `web_search`, `web_scraper`, `load_skill`, `load_tools`, `session_handshake`, `subagent`). The project's own workflows, components and agencies are always sent. `load_tools` names every other tool; the model loads the ones it needs, and their definitions go on every later request of the session:

```text
round 1: core definitions + load_tools ("Tools you can load: - git_ops: ...")
model:   load_tools {"names": "git_ops"}
round 2: core definitions + git_ops
```

The core set is `assets/policy/tool-schemas.yaml`:

```yaml
# ~/.kdeps/policy/tool-schemas.yaml overrides the built-in file
core:
  - bash_exec
  - read_file    # tools listed here always send their full definition
  - load_tools   # keep it: it is how the model reaches the rest
```

To send all definitions on each request, delete the file. The model can call a tool without a load of its definition.

## Compacting the saved conversation

kdeps saves only the user prompt and the final answer of each turn to the
session. It never saves the tool transcript of a turn. `/compact` summarizes the
older saved turns and keeps the last turns exactly. It runs when the session
has more than a small number of turns, at all sizes.

Auto-compaction between turns starts only when the saved conversation is more
than the token budget (`/model tool set compact-threshold <n>`). With many
short turns that use many tools, this can never occur. In this condition, use
`/compact` directly. Neither command changes the tool output of a turn that
runs. The in-flight window above sets the limit for that output.

If the summarizer leaves a heading empty, kdeps removes the heading before it
saves or shows the summary. That includes a bare `Summary` title and a `## Goal` with nothing
under it. The auto-compact line prints the first sentence of what remains,
not the heading.

The summary from that compact is not itself a reason to compact again. The
next one waits until real conversation, not the summary, has fallen outside
the kept window.

The compact budget sets how much recent conversation stays exactly. The default
of the compact budget and of the auto-compaction threshold is 3/4 of the known
context window of the model. It is not a fixed token count. A session on a
small local model and one on a
200K-token cloud model each get a good default. You do not have to set
`/model tool set compact-budget <n>` yourself. Switching models with `/model`
recomputes both immediately for the new model; an unrecognized model keeps a
conservative flat default.

kdeps never evaluates the conversation text that it summarizes as a kdeps
expression or template. This is also true if the text has double-curly-brace
syntax. Examples are kdeps expressions from the conversation, tags of a
different template language, or other text with this syntax. The summarizer
gets the text exactly as it is.

## Fold

`/fold` is a lighter and more frequent form of `/compact`. It does not wait
for the full compact threshold. It makes a checkpoint of the conversation each
time that a configurable quantity of *new* conversation collects after the last
checkpoint. The checkpoint is the same structured Goal/Progress/Decisions
summary, saved to [memory](/agent/memory-internals#checkpoint-summaries). It is
on by default (2000 tokens, and the last 5 checkpoints stay in the memory block
of the prompt). You do not need to set it up.

| Command | Effect |
|---|---|
| `/fold` | Show status: auto on/off, threshold, items cap, tokens accumulated since the last checkpoint |
| `/fold now` | Force a fold immediately, regardless of the threshold or the compaction budget |
| `/fold auto [on\|off]` | Show or set automatic folding (default: on) |
| `/fold threshold <n\|default>` | Token delta since the last checkpoint that triggers a fold (e.g. `4k`); `default` uses the `fold` event's value |
| `/fold items <n\|default>` | How many recent checkpoints stay in the active memory-prompt window |
| `/fold preset tight\|balanced\|loose` | Named threshold+items bundles (1000/3, the `fold` event's values, 8000/10) |

All four settings persist across sessions, the same way `/model tool set
compact-threshold`/`compact-budget` already do.

`/fold now` is a *forced* fold: it ignores the threshold and the compaction
budget entirely and always summarizes everything except the last few turns,
same as `/compact`. It returns nothing only when there are too few turns. In
this condition, all turns are in the part that it always keeps exactly. It
then prints the turn count and the number of recent turns that always stay.

`/fold` is a YAML command (`commands/fold.yaml`) with
[subcommands](/agent/commands#subcommands): the presets, their numbers and every
line it prints are in that file. Copy it to `~/.kdeps/commands/` to add a
preset of your own.

Both `/compact` and `/fold` reset `sent` and `generated` to the context that
remains. `sent` is that context (system prompt, memory, and the kept
history). `generated` is the model-written part still in it, including the
new summary. kdeps removes the old cumulative totals.
