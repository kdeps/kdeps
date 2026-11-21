# Efficiency enforcement

The efficiency governor stops a model that only reads (`ls`, `read_file`, `grep`, read-only `bash`) and makes no output. It is part of kdeps, not a prompt hint. A workflow `chat` turns it on with `harness.efficiency: true`, with or without `loop:` (see [Answer quality](/workflow/quality)).

```d2
direction: down
read: "model calls a read tool"
count: "reads in a row > limit?" {shape: diamond}
stop: "soft stop\ncall dropped\nread tools removed\nhidden note to the model"
out: "model calls an output tool\n(write, edit, bash, workflow)"
resume: "stop closed\nnext turn gets a reminder"
end: "turn ends after N stops\n(forced answer)" {shape: oval}
read -> count
count -> read: "no"
count -> stop: "yes"
stop -> out
stop -> end: "stops > limit"
out -> resume
```

## What it counts

| Class | Examples | Counted |
|---|---|---|
| read | `read_file`, `list_files`, `search_local`, web tools, `sql_query`, read-only `bash` (`ls`, `cat`, `git status`) | Yes. Reads in a row open a stop. |
| output | writes, edits, a `bash` command that changes state, workflow and component tools | Resets the read count and closes a stop. |
| state | `calculator`, `memory_*` | No. |

A turn also has a cap on all tool calls (`actions`) and on each category (`web`, `bash`, `file`, `code`).

## The hidden channel

The note goes to the model only. kdeps does not print it, and does not write it to history or sessions.

```text
[kdeps-efficiency] Soft stop 1/3. Why: 6 read-only calls in a row with no output. Needed: your next
call must be an OUTPUT action (write_file/edit_file/bash_exec) that changes something.
```

The same channel carries two more briefs:

- **Repeated call.** A call that the model repeats with the same result gets a brief: the call, the count, the result, and what to do instead. It uses the `stops` budget.
- **Failed call.** A failed tool call gets the error and the context to repair it: the lines around a failed `edit_file`, or the nearest folder of a missing path. A rejected model call gets the error, and the round runs again. These use the `failures` budget (default 5). After that, a failed tool result ends with `[TOOL FAILED]`.

Credential, billing, quota and unknown-model errors return at once. Temporary errors use backoff. A context overflow starts compaction.

## Prompt mode

The prompt mode sets how the harness sections go on the requests after the first one.

| Mode | Later requests |
|---|---|
| `auto` | One summary line per section, except on Anthropic and OpenAI (they cache the full prefix). The agent loop default. |
| `full` | The full sections. The workflow chat default. |
| `summary` | One summary line per section on every backend. |

The first request, and the first request after compaction, a fold or a model switch, always sends the full text. The summary line is the section's `summary:` field. In the agent loop, the model calls `instruct` with a section name to read the full text.

## Workflow chats

A workflow `chat` turns the governor on with `harness.efficiency` and sets the prompt mode with `harness.promptMode`. Both apply to its tool loop, and to a chat with `loop:`.

```yaml
# resources/fix.yaml
chat:
  context:
    prompt: "Fix the failing test in {{ get('path') }}."
  harness:
    sections: [accuracy]
    efficiency: true       # off by default in a workflow; limits come from the efficiency-* events
    promptMode: summary    # full by default in a workflow
  tools:
    list:
      - name: read_file
      - name: edit_file    # the governor applies only when the chat has an output tool
```

## Limits

| Setting | Meaning | Default |
|---|---|---|
| `reads` | read calls in a row before a stop | 6 |
| `actions` | tool calls per turn before a stop | 30 |
| `stops` | stops per turn before the turn ends | 3 |
| `failures` | failed calls briefed per turn | 5 |
| `tighten` | reads budget lost per stop | 1 |
| `web`, `bash`, `file`, `code` | calls per turn in that category | 5, 8, 10, 6 |

## Presets and events

The `frugal`, `balanced` and `thorough` [presets](/agent/events-presets#presets) set every value.

| Preset | reads | actions | stops | failures | tighten | web | bash | file | code |
|---|---|---|---|---|---|---|---|---|---|
| `frugal` | 4 | 20 | 2 | 3 | 1 | 3 | 5 | 6 | 4 |
| `balanced` | 6 | 30 | 3 | 5 | 1 | 5 | 8 | 10 | 6 |
| `thorough` | 12 | 60 | 4 | 8 | off | 8 | 15 | 20 | 10 |

Each value is an [event](/agent/events) in `~/.kdeps/events/`.

```yaml
# ~/.kdeps/events/efficiency-reads.yaml
name: efficiency-reads
on:
  rounds: 4        # read calls in a row before a soft stop
run: block         # the governor reads this event; it is not fired as an action
```

The governor is off unless `harness.efficiency: true`. It is also off when no output tool is offered.

## See also

- [Reactive LLM events](/agent/events) - the events that hold the limits
- [LLM resource](/workflow/resources/llm#efficiency) - `harness.efficiency` and `harness.promptMode`
