# Reactive LLM events

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
run: compact                    # same action as auto-compact, just a tighter trigger cadence
```

`auto-compact` is the full-context-window safety net: once the session gets close to filling the model's window, kdeps summarizes everything except the last few turns so the conversation can keep going. `fold` fires the same `compact` action on a tighter, more frequent cadence -- every time a couple thousand tokens of new conversation accumulate, well before `auto-compact` would ever need to fire. `fold` is not a separate, lighter mechanism today; the two events exist so the *trigger* is tunable independently even though the *action* is currently shared.

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

The other events, the actions they run, and the presets are their own pages.

| Topic | Page |
|---|---|
| Round counts, truncation, call budgets, memory caps | [Event guards](/agent/events-guards) |
| What an event runs, and hard vs soft enforcement | [Event actions](/agent/events-actions) |
| Turning events off, occurrence caps, presets | [Event presets](/agent/events-presets) |
