# Reactive LLM events

An event monitors one number. When the number crosses a limit, the event runs one action. kdeps uses events, not hardcoded Go constants, for the two automatic steps in a long conversation. The first step compacts old history. The second step puts recent history into a checkpoint.

Both events are YAML with built-in defaults. You can override them on each machine without a rebuild.

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

`auto-compact` is the safety net: near a full context window, kdeps summarizes everything except the last few turns. `fold` runs the same `compact` action more often, each time a couple thousand tokens of new conversation accumulate. They share an action. Separate events keep the triggers independently tunable.

## Overriding a built-in

Put a file with the same `name:` into `~/.kdeps/events/`. It fully replaces the built-in event. kdeps does not merge the fields. The trigger, the action, and `items` all come from your file:

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

`/model tool set compact-threshold` and `/fold threshold`, `/fold items`, `/fold preset` override the event default for the current session. The event's `tokens`, `tokensSinceCheckpoint` or `items` value supplies that default.

The other events, the actions they run, and the presets are their own pages.

| Topic | Page |
|---|---|
| Round counts, truncation, call budgets, memory caps | [Event guards](/agent/events-guards) |
| What an event runs, and hard vs soft enforcement | [Event actions](/agent/events-actions) |
| Turning events off, occurrence caps, presets | [Event presets](/agent/events-presets) |
