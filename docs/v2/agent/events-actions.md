# Event actions

An action is what an event runs. The check is Go control flow. The model does not choose it. The triggers stay on [Reactive LLM events](/agent/events).

## Actions

`run:` is not a free-form string. kdeps compares the action of each event with a registry of known actions. The registry uses the same pattern as events: built-in embeds and user overrides in `~/.kdeps`:

```yaml
# ~/.kdeps/actions/force_answer.yaml (built-in default -- shown for reference)
name: force_answer
kind: rounds     # the trigger shape this action is meant for: tokens, bytes, distinctCalls, items, or rounds
description: >-
  End the current tool-calling loop and force a final text answer from
  whatever was gathered, instead of letting the model keep spinning.
```

Nine actions ship, one per distinct behavior: `compact`, `truncate`, `reject`, `block`, `drop_oldest`, `cap`, `force_answer`, `fail_handshake`, `accept_last`. If the `run:` of an event has a different name (a typo, or a renamed or removed action), kdeps prints a startup warning to stderr. The warning gives the event and the action. A broken override always gives a warning.

## Behavior stays Go code

The behavior of each action is Go code, because the action has a real side effect. For example, it summarizes a conversation, truncates a string, or refuses a tool call. The registry is configurable: name, `kind` and `description`. `~/.kdeps/actions/`, `~/.kdeps/events/`, `~/.kdeps/harness/` and `~/.kdeps/themes/` share one embed-plus-override shape and are the source of truth for what happens and when. [konfig](./konfig.md) exports and imports all of them.

## Not every action is freely swappable

Actions are not freely swappable between events with the same trigger shape. `handshake-timeout`, `judge-max-rounds` and `subagent-max-rounds` (all `force_answer`) set a sub-loop's round budget and rely on the loop's round-exhaustion behavior. Changing their `run:` validates (the name exists) but changes nothing. The call site implements only its named action.

## Enforcement: hard vs. soft

No action needs the model's cooperation. Each is deterministic Go control flow: the system, not the model, decides to `compact` or `block`, usually because it removes the option before the turn of the model:

- **`block`/`reject`** -- `convergenceCache.trackCall()` and the file-size checks return an error *before* the real tool function runs. The shell command, file read, or web request never happens.
- **`force_answer`** (`identical-tool-calls`, `convergence-block`) -- `convergenceStop()` sets `cfg.Tools = nil` on the next LLM API request. When the request schema has no tools, the model cannot call a tool. This is a structural limit, not an instruction to the model.
- **`compact`** -- `shouldAutoCompact()`/`shouldFoldNow()` are plain boolean checks in `Loop.Run()`. Go code decides if and when this action starts, with no model input.
- **`truncate`/`drop_oldest`/`cap`** -- unconditional string/slice mutation in Go, applied every time regardless of model behavior.

## Real OS-level enforcement, not just Go control flow

`ToolStallTimeout` monitors the output of a tool that runs. If the tool gives no output for the timeout period, it cancels a real `context.Context`. Then `bash_exec` calls `killProcessGroup(cmd)`. This sends the equivalent of `SIGKILL` to the full process group, so a command that hangs and its child processes stop. The same cancellation stops the sub-loop of a judge after `judge_verdict` is final. Ctrl+C also uses it.

## The one genuinely soft layer

The only soft layer is the harness (`~/.kdeps/harness/*.yaml`: tool-use, memory, honesty, scope and accuracy rules): prompt text with no code checking compliance. That is the line for anything new. If it must hold regardless of the model, it is an event or action. If it is guidance, it is harness text.
