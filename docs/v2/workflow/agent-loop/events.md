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

A workflow project overrides events the same way with an `events/` folder next to its `resources/`. That layer wins over `~/.kdeps/events/` (see [Answer quality](/workflow/quality#the-workflow-ships-its-own-limits)).

A file in `~/.kdeps/events/` takes effect the next time the process starts.

The other events, the actions they run, and the presets are their own pages.

| Topic | Page |
|---|---|
| Round counts, truncation, call budgets, memory caps | [Event guards](/workflow/agent-loop/events-guards) |
| What an event runs, and hard vs soft enforcement | [Event actions](/workflow/agent-loop/events-actions) |
| Turning events off, occurrence caps, presets | [Event presets](/workflow/agent-loop/events-presets) |
