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

Nine actions ship, one per distinct behavior: `compact`, `truncate`, `reject`, `block`, `drop_oldest`, `cap`, `force_answer`, `fail_handshake`, `accept_last`. An event whose `run:` names anything else (a typo, a renamed or removed action) prints a startup warning to stderr naming the event and the action. A broken override is never a silent no-op.

## Behavior stays Go code

Each action's behavior is Go code, since it performs a real side effect: summarizing a conversation, truncating a string, refusing a tool call. The registry is configurable: name, `kind` and `description`. `~/.kdeps/actions/`, `~/.kdeps/events/`, `~/.kdeps/harness/` and `~/.kdeps/themes/` share one embed-plus-override shape and are the source of truth for what happens and when. [konfig](./konfig.md) exports and imports all of them.

## Not every action is freely swappable

Actions are not freely swappable between events with the same trigger shape. `handshake-timeout` and `judge-max-rounds` (both `force_answer`) set a sub-loop's round budget and rely on the loop's round-exhaustion behavior. Changing their `run:` validates (the name exists) but changes nothing; the call site implements only its named action.

## Enforcement: hard vs. soft

No action needs the model's cooperation. Each is deterministic Go control flow: the system, not the model, decides to `compact` or `block`, usually by removing the option before the model's turn:

- **`block`/`reject`** -- `convergenceCache.trackCall()` and the file-size checks return an error *before* the real tool function runs. The shell command, file read, or web request never happens.
- **`force_answer`** (`identical-tool-calls`, `convergence-block`) -- `convergenceStop()` sets `cfg.Tools = nil` on the next LLM API request. Once tools are not in the request schema, the model is structurally unable to call one -- not persuaded not to, incapable of it.
- **`compact`** -- `shouldAutoCompact()`/`shouldFoldNow()` are plain boolean checks in `Loop.Run()`; Go decides whether and when this fires, with no model input.
- **`truncate`/`drop_oldest`/`cap`** -- unconditional string/slice mutation in Go, applied every time regardless of model behavior.

## Real OS-level enforcement, not just Go control flow

`ToolStallTimeout` watches a running tool's output for wall-clock silence. Past the timeout it cancels a real `context.Context`; `bash_exec` then calls `killProcessGroup(cmd)`, a `SIGKILL`-equivalent to the whole process group, so a hung command and its children die. The same cancellation ends a judge's sub-loop once `judge_verdict` settles, and backs Ctrl+C.

## The one genuinely soft layer

The only soft layer is the harness (`~/.kdeps/harness/*.yaml`: tool-use, memory, honesty, scope and accuracy rules): prompt text with no code checking compliance. That is the line for anything new. If it must hold regardless of the model, it is an event or action; if it is guidance, it is harness text.
