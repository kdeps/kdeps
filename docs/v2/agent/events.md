# Reactive LLM events

*Applies to agent mode.*

An event is a small rule: watch one number, and once it crosses a line, run one action. kdeps uses this for the two things that happen automatically during a long conversation -- compacting old history and folding recent history into a checkpoint -- instead of hardcoding "30000" and "2000" somewhere in the Go source. Both are plain YAML, both ship with sane built-in defaults, and both can be overridden per-machine without a rebuild.

```text
measure tokens --> compare to threshold --> fire action (compact / fold)
```

## The two built-in events

```yaml
# ~/.kdeps/events/auto-compact.yaml (built-in default -- shown for reference,
# not present on disk until you override it)
name: auto-compact
on:
  tokens: 30000            # flat fallback for a model with an unknown context window
  ctxWindowFraction: 0.75  # preferred once the model's window IS known: fire at 75% full
  minTurns: 4              # never fire before at least 4 turns exist
run: compact                # summarize everything except the last few turns
```

```yaml
# ~/.kdeps/events/fold.yaml
name: fold
on:
  tokensSinceCheckpoint: 2000  # tokens accumulated since the last checkpoint
  minTurns: 4
items: 5                       # how many recent checkpoints stay in the memory-prompt window
run: fold                       # a lighter, more frequent pass than auto-compact
```

`auto-compact` is the full-context-window safety net: once the session gets close to filling the model's window, kdeps summarizes everything except the last few turns so the conversation can keep going. `fold` is a tighter, more frequent pass -- it archives a checkpoint every time a couple thousand tokens of new conversation accumulate, well before `auto-compact` would ever need to fire.

## Overriding a built-in

Drop a file with the same `name:` into `~/.kdeps/events/` and it replaces the built-in outright (not merged field-by-field -- the whole trigger, action, and `items` come from your file):

```yaml
# ~/.kdeps/events/auto-compact.yaml
name: auto-compact
on:
  tokens: 60000       # wait longer before compacting
  minTurns: 8
run: compact
```

No restart needed for `/konfig import`, `kdeps konfig import`, or `--konfig <path>` -- see [konfig](./konfig.md). A file you drop into `~/.kdeps/events/` by hand takes effect the next time the process starts.

## Relationship to `/model tool set` and `/fold`

The `/model tool set` knobs `compact-threshold` and the `/fold threshold`/`/fold items`/`/fold preset` commands still work exactly as before -- they set an explicit override for the *current session*, on top of whatever the event's own default is. An event's `tokens`/`tokensSinceCheckpoint`/`items` value is where that default comes from; it is no longer a hardcoded Go constant duplicated at every call site that needed it.

## Round-count events

The same `{name, on, run}` shape also covers the loop's stuck-loop and task-budget guards -- these fire on a *count of rounds*, not a token measurement, so their `on:` clause uses `rounds:` instead of `tokens:`:

```yaml
# ~/.kdeps/events/identical-tool-calls.yaml
name: identical-tool-calls
on:
  rounds: 3   # how many times in a row the model may issue the exact same tool call
run: force_answer
```

| Event | Fires after | Action |
|---|---|---|
| `identical-tool-calls` | the model issues the exact same tool call this many times in a row | ends the turn as a stuck loop |
| `convergence-block` | this many consecutive rounds end in a convergence-blocked tool call (the model kept trying once a web/bash/file/code budget ran out) | strips every tool and forces a text answer |
| `task-round-budget` | a single goal-directed task spends this many tool rounds | force-closes the task |
| `unproductive-rounds` | this many consecutive rounds produce no new tool result or state change | force-closes the task and fails it forward |
| `handshake-timeout` | a session-handshake grounding challenge (see [session-integrity handshake](/agent/tools#session-integrity-handshake)) spends this many tool rounds without resolving | fails the handshake |

Every one of these has a sensible floor of 1 built in: an override file that omits `rounds:`, or a corrupted one, never accidentally disables the guard by resolving to 0 (which would otherwise fire on the very first round).

## Truncation events

The same shape covers the loop's token-economy limits -- these fire on a *measured byte length*, so their `on:` clause uses `bytes:` instead:

```yaml
# ~/.kdeps/events/tool-result-truncate.yaml
name: tool-result-truncate
on:
  bytes: 16384   # any single tool result past this size is truncated
run: truncate
```

| Event | Fires when | Action |
|---|---|---|
| `tool-result-truncate` | a single tool result fed back to the LLM exceeds this size | truncates on a line boundary, appends a marker |
| `tool-error-truncate` | a tool's failure text exceeds this size | truncates before display and before feeding it back to the LLM |
| `force-answer-digest` | the gathered-output digest inlined into a forced answer exceeds this size | truncates the digest |
| `history-window-trim` | the in-flight tool-loop message array exceeds this size | drops the oldest complete tool round-trips |
| `file-read-limit` | a file a builtin tool is about to read (or `write_file`'s own content) exceeds this size | rejects the call with an error instead of reading/writing it |

Like the round-count cluster, each has its own sensible fallback (matching its built-in default) if the event registry is ever unavailable -- an override that omits `bytes:` never resolves to "truncate to nothing" or "reject every file."

## Call-budget events

The convergence caps on `web_search`/`web_scraper`, `bash_exec`, `read_file`/`list_files`, and `search_local`/`code_search` (see [tools](/agent/tools)) are also events -- these fire on a *distinct-call count per turn*, so their `on:` clause uses `distinctCalls:`:

```yaml
# ~/.kdeps/events/web-call-budget.yaml
name: web-call-budget
on:
  distinctCalls: 20   # DISTINCT web_search + web_scraper calls this turn -- a repeat never counts twice
run: block
```

| Event | Caps | Default |
|---|---|---|
| `web-call-budget` | distinct `web_search`/`web_scraper` calls per turn (they share one budget) | 20 |
| `bash-call-budget` | distinct `bash_exec` commands per turn | 50 |
| `file-call-budget` | distinct `read_file`/`list_files` paths per turn | 80 |
| `code-call-budget` | distinct `search_local`/`code_search` queries per turn | 30 |

These four already had a per-session override path before events existed -- `/model tool set web-limit <n>` (and `bash-limit`/`file-limit`/`code-limit`) still work exactly as before, on top of whatever the event's own default is, the same relationship `/fold threshold` has with the `fold` event.

## Status

Sixteen events ship today: `auto-compact`, `fold`, `identical-tool-calls`, `convergence-block`, `task-round-budget`, `unproductive-rounds`, `handshake-timeout`, `tool-result-truncate`, `tool-error-truncate`, `force-answer-digest`, `history-window-trim`, `file-read-limit`, `web-call-budget`, `bash-call-budget`, `file-call-budget`, and `code-call-budget`. All are read-only from the REPL (there is no `/event set` command yet -- edit the YAML file directly, the same way a custom `/theme` or harness section is authored). A judge-cluster slice (judge round/iteration caps) may follow the same pattern later.
