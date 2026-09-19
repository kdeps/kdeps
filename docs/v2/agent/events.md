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

## Status

Seven events ship today: `auto-compact`, `fold`, `identical-tool-calls`, `convergence-block`, `task-round-budget`, `unproductive-rounds`, and `handshake-timeout`. All are read-only from the REPL (there is no `/event set` command yet -- edit the YAML file directly, the same way a custom `/theme` or harness section is authored). Later phases may cover more of the hardcoded truncation and call-budget limits the same way; see the design notes for the full list under consideration.
