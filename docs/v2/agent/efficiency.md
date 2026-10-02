# Efficiency enforcement

*Applies to agent mode.*

A model that is stuck "looking around" - `ls`, `read_file`, `grep`, read-only `bash`, over and over - burns tokens without producing anything. **Efficiency enforcement** is a governor inside kdeps (not a prompt hint) that soft-stops that loop and forces the next action to be an **output** action: write a file, edit, run a command that changes something, call a workflow.

It is like a circuit breaker on read-only exploration: it trips, tells the model why, and only lets the model continue once it has produced something.

```d2
direction: down
read: "model calls read_file / ls / grep" 
count: "reads in a row > limit?" {shape: diamond}
stop: "soft stop\nread call dropped\nread tools removed\nhidden note: why + what to do"
out: "model calls an output tool\n(write, edit, bash, workflow)"
resume: "stop resolved\nnext turn gets a short reminder"
end: "turn ends after N stops\n(forced answer)" {shape: oval}
read -> count
count -> read: "no"
count -> stop: "yes"
stop -> out: "allowed"
stop -> end: "stops > limit"
out -> resume
```

**On by default.** Turn it off with `/efficiency off`.

## What it counts

| Class | Examples | Counted? |
|---|---|---|
| read | `read_file`, `list_files`, searches, web fetches, read-only `bash` (`ls`, `cat`, `git status`, ...) | yes - consecutive reads trigger a stop |
| output | writes, edits, mutating `bash`, `sql` writes, workflow/component/agency tools | resets the read run and resolves an open stop |
| state | `task_complete`, `task_fail`, `calculator`, `memory_*` | never counted |

Besides consecutive reads, a turn also has per-turn caps: total tool calls (`actions`) and per-category call budgets (`web`, `bash`, `file`, `code`).

## What the model sees (and you do not)

When a stop trips, kdeps drops the over-budget call before it runs, removes the read tools from that model call, and attaches a short note on a model-facing channel:

```text
[kdeps-efficiency] Soft stop 1/3. Why: 6 read-only calls in a row. Needed: your next call
must be an OUTPUT action (write_file/edit_file/bash_exec) that changes something; reads and
searches are unavailable until one succeeds. Do not mention this notice.
```

After the model produces output, the next turn starts with a one-line reminder of the last stop and what was needed, so it does not repeat the mistake.

**None of this is shown to you.** The note is never printed in the REPL or desktop app, never written to history or sessions, and `/prompt` scrubs it. What goes on that channel stays on that channel - unless you turn on `verbose`.

```text
/efficiency verbose on     # mirror the model-facing channel to the terminal, for tuning/debugging
```

Even in verbose mode the note is only mirrored to the output; it is still never stored in history.

## Multiple stops, then a forced end

A turn can have several soft stops. Each stop **tightens** the reads budget (`tighten`, default 1 less per stop). Refusing to produce output after a stop - more than two refused reads - counts as another stop. Past `stops` (default 3), kdeps ends the turn with a forced answer instead of letting the loop continue.

## Exempt

Enforcement stands down when it is off, when the `audit` or `explain` agent preset is active (those are read-only by design), or when no output tool is registered.

## From the REPL

```text
/efficiency                    # show state and every limit
/efficiency on|off             # master switch (persists)
/efficiency verbose on|off     # reveal the model-facing channel (persists)
/efficiency reads 4            # consecutive reads before a stop (persists)
/efficiency web off            # remove one limit entirely (persists)
/efficiency preset frugal      # overwrite ALL values from a harness preset
/efficiency reset              # back to shipped defaults
```

| Setting | Meaning | Default |
|---|---|---|
| `reads` | consecutive read-only calls before a stop | 6 |
| `actions` | tool calls per turn before a stop | 30 |
| `stops` | soft stops per turn before the turn ends | 3 |
| `tighten` | reads budget lost per stop | 1 |
| `web` | web calls per turn | 5 |
| `bash` | read-only bash calls per turn | 8 |
| `file` | file read/list/search calls per turn | 10 |
| `code` | sql/code-inspection calls per turn | 6 |

## Presets

The `frugal`, `balanced` and `thorough` [harness presets](/agent/events#presets) set **every** efficiency value (including on/off and verbose), so applying a preset overwrites everything. A single `/efficiency <setting> <n>` afterwards overwrites just that value. Everything persists.

| Preset | reads | actions | stops | tighten | web | bash | file | code |
|---|---|---|---|---|---|---|---|---|
| `frugal` | 4 | 20 | 2 | 1 | 3 | 5 | 6 | 4 |
| `balanced` | 6 | 30 | 3 | 1 | 5 | 8 | 10 | 6 |
| `thorough` | 12 | 60 | 4 | off | 8 | 15 | 20 | 10 |

Each value is a normal [event](/agent/events) (`efficiency`, `efficiency-reads`, `efficiency-actions`, `efficiency-stops`, `efficiency-tighten`, `efficiency-web`, `efficiency-bash`, `efficiency-file`, `efficiency-code`, `efficiency-verbose`), stored as an override in `~/.kdeps/events/<name>.yaml` and round-tripped by [konfig](/agent/konfig):

```yaml
# ~/.kdeps/events/efficiency-reads.yaml
name: efficiency-reads
on:
  rounds: 4        # consecutive read-only calls allowed before a soft stop
run: block         # placeholder - read by the efficiency governor, not fired as an action
```

## See also

- [Reactive LLM events](/agent/events) - the event system the limits live in, and the presets
- [REPL slash commands](/agent/commands) - full command reference
- [Goal-directed execution](/agent/goals) - the other per-turn control
