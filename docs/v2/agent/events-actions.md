# Event actions

An action is what an event runs. The check is Go control flow. The model does not choose it. The triggers stay on [Reactive LLM events](/agent/events).

## Actions

`run:` is not a free-form string -- every event's declared action is checked against a registry of known actions, the same built-in-embed + `~/.kdeps` user-override pattern as events themselves:

```yaml
# ~/.kdeps/actions/force_answer.yaml (built-in default -- shown for reference)
name: force_answer
kind: rounds     # the trigger shape this action is meant for: tokens, bytes, distinctCalls, items, or rounds
description: >-
  End the current tool-calling loop and force a final text answer from
  whatever was gathered, instead of letting the model keep spinning.
```

Ten actions ship today, one per distinct behavior in the codebase: `compact`, `truncate`, `reject`, `block`, `drop_oldest`, `cap`, `force_answer`, `fail_task`, `fail_handshake`, `accept_last`. If an event's `run:` names anything else -- a typo, or a name that was renamed or removed -- kdeps prints a warning to stderr at startup naming the event and the unrecognized action, so a broken override is loud, never a silent no-op.

## Behavior stays Go code

The action's actual behavior stays Go code (these perform real side effects -- summarizing a conversation, truncating a string, refusing a tool call -- not something a YAML file alone can express). The registry is what is configurable: the name, its `kind`, and its `description`, which is what makes `~/.kdeps/actions/*.yaml`, `~/.kdeps/events/*.yaml`, `~/.kdeps/harness/*.yaml`, and `~/.kdeps/themes/*.yaml` the complete, consistent source of truth for "what can happen and when" -- every one of them the same embed-plus-override shape, every one covered by [konfig](./konfig.md) export/import.

## Not every action is freely swappable

Not every action is freely swappable onto any event with the same trigger shape: `task-round-budget`/`unproductive-rounds` (both `fail_task`) actually drive a 4-step escalation ladder in the goal-enforcement subsystem (reanchor, narrow tools, force close, then fail forward), not a single function call, and `handshake-timeout`/`judge-max-rounds` (both `force_answer`) work by setting a sub-loop's round budget and relying on the loop's own generic round-exhaustion behavior rather than a distinct dispatchable step. Renaming these two pairs' `run:` to something else would be validated (the name exists) but would not change what actually happens -- the call site only implements the one action it already names.

## Enforcement: hard vs. soft

None of this asks the model to cooperate. Every action is deterministic Go control flow that runs regardless of what the model wants -- the model does not decide to `compact` or `block`; the system does it around the model, usually by removing the option before the model ever gets a turn:

- **`block`/`reject`** -- `convergenceCache.trackCall()` and the file-size checks return an error *before* the real tool function runs. The shell command, file read, or web request never happens.
- **`force_answer`** (`identical-tool-calls`, `convergence-block`) -- `convergenceStop()` sets `cfg.Tools = nil` on the next LLM API request. Once tools are not in the request schema, the model is structurally unable to call one -- not persuaded not to, incapable of it.
- **`compact`** -- `shouldAutoCompact()`/`shouldFoldNow()` are plain boolean checks in `Loop.Run()`; Go decides whether and when this fires, with no model input.
- **`fail_task`** (the goal-enforcement escalation ladder: reanchor -> narrow -> force close -> fail forward) -- the force-close step (`refuseOffTask`) intercepts every non-task-state tool call and substitutes a refusal string for the real result; the fail-forward step (`failForward` -> `Goal.Advance`) is called automatically at the end of every round from `enforceGoalProgress`, moving the task cursor whether or not the model ever calls `task_fail`. No step in the ladder is a suggestion.
- **`truncate`/`drop_oldest`/`cap`** -- unconditional string/slice mutation in Go, applied every time regardless of model behavior.

## Real OS-level enforcement, not just Go control flow

`ToolStallTimeout` runs a background goroutine that watches wall-clock silence on a running tool's output; past the timeout, it cancels a real `context.Context`, and `bash_exec` selects on that cancellation to call `killProcessGroup(cmd)` -- an actual `SIGKILL`-equivalent to the whole process group, so a hung shell command (and anything it spawned) is killed outright. The same context-cancellation path ends a judge's ephemeral sub-loop the instant `judge_verdict` settles, and backs Ctrl+C.

## The one genuinely soft layer

The one genuinely soft layer is the harness (`~/.kdeps/harness/*.yaml` -- tool-use rules, memory rules, honesty/scope/accuracy sections): prompt text asking the model to behave a certain way, with nothing in code checking whether it actually did. That is the dividing line for anything new: if it must be true regardless of what the model does, it is an event/action; if it is guidance the model is expected to follow, it is harness text.
