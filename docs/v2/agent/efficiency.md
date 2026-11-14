# Efficiency enforcement

A model stuck "looking around" (`ls`, `read_file`, `grep`, read-only `bash`, over and over) burns tokens without output. **Efficiency enforcement** is a kdeps governor, not a prompt hint. It soft-stops that loop and forces the next action to produce output: write a file, edit, run a changing command, or call a workflow.

It is like a circuit breaker on read-only exploration. It stops the model and tells the model why. The model can continue only after it makes some output.

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
| state | `calculator`, `memory_*` | never counted |

Besides consecutive reads, a turn also has per-turn caps: total tool calls (`actions`) and per-category call budgets (`web`, `bash`, `file`, `code`).

## What the model sees (and you do not)

When a stop occurs, kdeps removes the call that is over budget before it runs. It also removes the read tools from that model call. Then it attaches a short note on a channel that only the model sees:

```text
[kdeps-efficiency] Soft stop 1/3. Why: 6 read-only calls in a row. Needed: your next call
must be an OUTPUT action (write_file/edit_file/bash_exec) that changes something; reads and
searches are unavailable until one succeeds. Do not mention this notice.
```

After the model makes output, the next turn starts with a one-line reminder. The reminder gives the last stop and the necessary action, so the model does not make the same mistake again.

**You do not see any of this.** kdeps never prints the note in the REPL or desktop app, and never writes it to history or sessions. `/prompt` removes it. Data on that channel stays on that channel, unless you turn on `verbose`.

```text
/efficiency verbose on     # mirror the model-facing channel to the terminal, for tuning/debugging
```

In verbose mode, kdeps only copies the note to the output. It never stores the note in history.

## Repeated identical calls

A model repeating the same tool call and getting the same result is not progressing. The stuck-loop guard hands this to the efficiency manager as a soft stop. The model gets a hidden brief (which call it repeated, how often, what came back, what to do instead) and retries.

```text
[kdeps-efficiency] Soft stop 1/3. Why: you called bash_exec ({"command":"go test"}) 3 times in a row
and got the same result each time ("FAIL ..."). Repeating an identical call cannot change that
outcome, so no progress was made. Needed: do something different - change the arguments, use another
tool or approach, or fix the cause the result points to - then retry the task. Do not mention this notice.
```

Like every efficiency note, the brief is never shown to you (unless `verbose` is on) and never stored. It uses the same `stops` budget. When that budget is empty, or when efficiency enforcement is off, the turn ends with the plain "The model repeated the same ..." notice.

## Failed tool calls and failed model calls

kdeps stops a failed call itself. The model does not have to see the failure.

A tool call can fail. Examples are an edit that did not match, a missing path, bad arguments, an unknown tool, or a denied call. A backend can also reject the model call. In these conditions, kdeps does a soft stop. It sends the error and the context to repair it on the hidden channel. Then the model tries again.

```d2
direction: down
call: "model calls a tool" {shape: oval}
failed: "call failed?" {shape: diamond}
ok: "result goes to history"
brief: "soft stop\nhidden brief: error + context\n(file lines / folder listing / retry example)"
budget: "failures left this turn?" {shape: diamond}
banner: "plain [TOOL FAILED] banner\nin the tool result"
retry: "model retries with the new information"
call -> failed
failed -> ok: "no"
failed -> budget: "yes"
budget -> brief: "yes"
budget -> banner: "no"
brief -> retry
retry -> call
```

## What the brief carries

What the brief carries depends on the failure:

| Failure | Extra context in the brief |
|---|---|
| `edit_file` did not apply | the file's current lines around the spot the edit aimed at (20 lines either side, numbered), plus the read-first retry example |
| a path does not exist | the entries of the nearest existing folder, so the model can pick the real name |
| any other tool error | the error, the call's arguments and a retry example |
| the model call was rejected (bad request, malformed tool call) | the error; the round is re-run |

```text
[kdeps-efficiency] Soft stop (failure 1/5). Your last call failed and nothing changed: edit_file
({"file_path":"/app/main.go","line":44,"match":"cfg",...}). Error: "edit_file: line 44 does not contain 'cfg'"

Current content of /app/main.go, lines 40-80 of 212:
40	func main() {
41		cfg := load()
...

Needed: use this to fix the cause and call again now - correct the arguments, the path or the text
you match on, or use another tool. If it cannot be done, say plainly in your answer that this step
failed; do NOT report it as done. Do not mention this notice.
```

## The tool result in history

History keeps only the raw error in the tool result (APIs require a result per call). The brief, file lines and retry example never reach history, the REPL or the desktop app unless `verbose` is on. The tool line still shows the failure.

## Errors the model cannot fix

kdeps does not use this procedure for model-call errors that the model cannot repair. Temporary errors (rate limits, 5xx, network) try again with backoff. kdeps compacts the context when it overflows. Credential, billing, quota, and unknown-model errors return to you immediately.

## The failures budget

Failures have their own per-turn budget, `failures` (default 5), separate from `stops`. When this budget is empty, tool calls get the plain `[TOOL FAILED]` banner, and a model call that fails returns its error. This also applies when the turn has no more rounds first. For this reason, a turn never ends silently on a hidden error.

## Multiple stops, then a forced end

A turn can have several soft stops. Each stop tightens the reads budget (`tighten`, default 1 less per stop). Refusing to produce output after a stop - more than two refused reads - counts as another stop. Past `stops` (default 3), kdeps ends the turn with a forced answer instead of letting the loop continue.

## Exempt

Enforcement stops in these conditions: it is off, the `audit` or `explain` agent preset is active (these presets are read-only by design), or no output tool is registered.

## Harness prompt

`/efficiency prompt` sets what occurs to the harness after kdeps sends its full text one time.

```text
/efficiency prompt                 # show auto, full, or summary
/efficiency prompt auto            # summary only where the prompt is billed every request (default)
/efficiency prompt full            # always send the full sections
/efficiency prompt summary         # summary after the first send, on every backend
```

The first request of a session, and the first request after compaction, a fold, or a model switch, always sends the full harness. Later requests in `summary` mode, and in `auto` mode on a backend other than Anthropic or OpenAI, send one line per section. That line is the hand-written `summary:` field in the section's harness YAML. The model calls `instruct` (alias `help`) with the section name to read the full text.

`auto` leaves Anthropic and OpenAI on the full text. They cache a stable system prefix, so swapping in a summary barely saves anything and invalidates that cache. Ollama, llamafile, GGUF, M365, and the other backends pay for the prompt on every request, so the summary is the saving. The choice persists in `~/.kdeps/agent-loop-settings.yaml` (`prompt_mode`).

## From the REPL

```text
/efficiency                    # show state and every limit
/efficiency on|off             # master switch (persists)
/efficiency verbose on|off     # reveal the model-facing channel (persists)
/efficiency prompt auto|full|summary   # harness text after the first send (persists)
/efficiency reads 4            # consecutive reads before a stop (persists)
/efficiency failures 8         # failed calls briefed and retried per turn (persists)
/efficiency web off            # remove one limit entirely (persists)
/efficiency preset frugal      # overwrite ALL values from a harness preset
/efficiency reset              # back to shipped defaults
```

| Setting | Meaning | Default |
|---|---|---|
| `reads` | consecutive read-only calls before a stop | 6 |
| `actions` | tool calls per turn before a stop | 30 |
| `stops` | soft stops per turn before the turn ends | 3 |
| `failures` | failed tool/LLM calls briefed and retried per turn | 5 |
| `tighten` | reads budget lost per stop | 1 |
| `web` | web calls per turn | 5 |
| `bash` | read-only bash calls per turn | 8 |
| `file` | file read/list/search calls per turn | 10 |
| `code` | sql/code-inspection calls per turn | 6 |

## Presets

The `frugal`, `balanced` and `thorough` [harness presets](/agent/events-presets#presets) set every efficiency value (including on/off and verbose), so applying a preset overwrites everything. After that, one `/efficiency <setting> <n>` overwrites only that value. Everything persists.

| Preset | reads | actions | stops | failures | tighten | web | bash | file | code |
|---|---|---|---|---|---|---|---|---|---|
| `frugal` | 4 | 20 | 2 | 3 | 1 | 3 | 5 | 6 | 4 |
| `balanced` | 6 | 30 | 3 | 5 | 1 | 5 | 8 | 10 | 6 |
| `thorough` | 12 | 60 | 4 | 8 | off | 8 | 15 | 20 | 10 |

Each value is a normal [event](/agent/events). kdeps stores it as an override in `~/.kdeps/events/<name>.yaml`, and [konfig](/agent/konfig) exports and imports it. The events are `efficiency`, `efficiency-reads`, `efficiency-actions`, `efficiency-stops`, `efficiency-failures`, `efficiency-tighten`, `efficiency-web`, `efficiency-bash`, `efficiency-file`, `efficiency-code`, and `efficiency-verbose`:

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
