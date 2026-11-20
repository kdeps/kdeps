# Status line

The REPL status line shows the model, the token counts, and the call budgets. The REPL uses one fixed color palette. The other REPL behaviors are on [REPL features](/agent/repl).

## Model name

The status line shows the model name in full. Change that with `/model name`:

```text
/model name              show the current mode
/model name show         show the literal model name
/model name hide         omit the model name from the modeline
/model name abbreviate   show initials, such as CS5 for claude-sonnet-5
/model name auto         show the full name (the factory setting)
```

`/model name <mode>` persists to `~/.kdeps/agent-loop-settings.yaml`.

## The spinner

The spinner shows the glyph and the token counter. It does not show a word such as generating or thinking.

## Context-path status line

If kdeps knows the context window of the current model, a line above the session counter shows the parts of the prompt for this turn:

```
[turn 12.4k/200k | sys 8.1k | mem 1.2k | hist 2.4k | bash_exec 700]
[session sent 1.2m | generated 30 | total 1.2m | this turn +47.2k sent +400 generated]
```

`turn A/B` is this prompt's size over the model's context window. The groups after it are that same prompt, split: `sys` system prompt, `mem` memory, `hist` conversation history, and one entry per tool name. Repeated calls to the same tool add to that one number. Past 5 groups the oldest collapse into `+N`. The line resets with each new prompt.

## The session row

The `session` row is cumulative and never measured against the window. Only `turn` is.

| Field | Meaning |
|---|---|
| `sent` | prompt tokens sent this session. History is resent every round, so it outgrows the window; it is not context fullness |
| `generated` | tokens the model wrote this session, including reasoning |
| `total` | `sent` + `generated` |
| `this turn` | the sent/generated delta since your last message; shown once the turn spends anything |
| `web`, `sh`, `file`, `src` | `used/limit` of a call budget, shown after its first call |

- `sent` and `generated` update when a reply streams and when a tool runs.
- `/compact` and `/fold` reset both to the remaining context: `sent` is that context, `generated` the model-written part of it.
- The two rows are one frame, rewritten in place each tick. Rows wider than the terminal are cut. Non-terminal output reprints the rows only when numbers change.

## The turn figure

`turn` is the prompt size of the latest call: the provider's count when the backend reports one, else the size of the messages sent. The parts (`sys`, `mem`, `hist`, tools) are pre-call estimates scaled to that total. The rest (your message, tool schemas, framing) shows as `other`. Before a turn's first call, parts are plain estimates. The modeline `used/limit` shows the same figure in the same format (`20.7k/128.0k`).

