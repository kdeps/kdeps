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
| `judge-max-rounds` | a single [judge](./judges.md)'s review spends this many tool rounds | ends that judge's review and forces its verdict |
| `judge-iterations` | the revise-and-rejudge loop retries this many times after a judge rejection | accepts the last response regardless of outcome -- a judge panel must never block a turn indefinitely |

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

## Memory events

The [persistent memory](./memory.md) subsystem's injection/traversal caps are events too. `memory-prompt-limit` fires on a token measurement like `auto-compact`/`fold`; the rest are plain counts with no trigger condition at all -- they always apply, so they set `items:` directly instead of an `on:` clause:

```yaml
# ~/.kdeps/events/memory-focus-max.yaml
name: memory-focus-max
items: 5   # how many prompt-relevant entries are force-kept when memory is truncated
run: cap
```

| Event | Caps | Default |
|---|---|---|
| `memory-prompt-limit` | tokens of memory content injected into the system preamble each turn | 500 |
| `memory-keys-limit` | key names listed in the `<memory-keys>` preamble block | 100 |
| `memory-focus-max` | prompt-relevant entries force-kept when memory is truncated | 5 |
| `memory-chain-max` | entries the active task chain force-keeps (its nearest ancestors) | 8 |
| `rel-memory-limit` | base memory rows fed into a `memory_query` join, bounding its worst case | 500 |

## Actions

`run:` isn't a free-form string -- every event's declared action is checked against a registry of known actions, the same built-in-embed + `~/.kdeps` user-override pattern as events themselves:

```yaml
# ~/.kdeps/actions/force_answer.yaml (built-in default -- shown for reference)
name: force_answer
kind: rounds     # the trigger shape this action is meant for: tokens, bytes, distinctCalls, items, or rounds
description: >-
  End the current tool-calling loop and force a final text answer from
  whatever was gathered, instead of letting the model keep spinning.
```

Ten actions ship today, one per distinct behavior in the codebase: `compact`, `truncate`, `reject`, `block`, `drop_oldest`, `cap`, `force_answer`, `fail_task`, `fail_handshake`, `accept_last`. If an event's `run:` names anything else -- a typo, or a name that was renamed or removed -- kdeps prints a warning to stderr at startup naming the event and the unrecognized action, so a broken override is loud, never a silent no-op.

The action's actual behavior stays Go code (these perform real side effects -- summarizing a conversation, truncating a string, refusing a tool call -- not something a YAML file alone can express). The registry is what's configurable: the name, its `kind`, and its `description`, which is what makes `~/.kdeps/actions/*.yaml`, `~/.kdeps/events/*.yaml`, `~/.kdeps/harness/*.yaml`, and `~/.kdeps/themes/*.yaml` the complete, consistent source of truth for "what can happen and when" -- every one of them the same embed-plus-override shape, every one covered by [konfig](./konfig.md) export/import.

Not every action is freely swappable onto any event with the same trigger shape: `task-round-budget`/`unproductive-rounds` (both `fail_task`) actually drive a 4-step escalation ladder in the goal-enforcement subsystem (reanchor, narrow tools, force close, then fail forward), not a single function call, and `handshake-timeout`/`judge-max-rounds` (both `force_answer`) work by setting a sub-loop's round budget and relying on the loop's own generic round-exhaustion behavior rather than a distinct dispatchable step. Renaming these two pairs' `run:` to something else would be validated (the name exists) but wouldn't change what actually happens -- the call site only implements the one action it already names.

## Enforcement: hard vs. soft

None of this asks the model to cooperate. Every action is deterministic Go control flow that runs regardless of what the model wants -- the model doesn't decide to `compact` or `block`; the system does it around the model, usually by removing the option before the model ever gets a turn:

- **`block`/`reject`** -- `convergenceCache.trackCall()` and the file-size checks return an error *before* the real tool function runs. The shell command, file read, or web request never happens.
- **`force_answer`** (`identical-tool-calls`, `convergence-block`) -- `convergenceStop()` sets `cfg.Tools = nil` on the next LLM API request. Once tools aren't in the request schema, the model is structurally unable to call one -- not persuaded not to, incapable of it.
- **`compact`** -- `shouldAutoCompact()`/`shouldFoldNow()` are plain boolean checks in `Loop.Run()`; Go decides whether and when this fires, with no model input.
- **`fail_task`** (the goal-enforcement escalation ladder: reanchor -> narrow -> force close -> fail forward) -- the force-close step (`refuseOffTask`) intercepts every non-task-state tool call and substitutes a refusal string for the real result; the fail-forward step (`failForward` -> `Goal.Advance`) is called automatically at the end of every round from `enforceGoalProgress`, moving the task cursor whether or not the model ever calls `task_fail`. No step in the ladder is a suggestion.
- **`truncate`/`drop_oldest`/`cap`** -- unconditional string/slice mutation in Go, applied every time regardless of model behavior.

**Real OS-level enforcement, not just Go control flow:** `ToolStallTimeout` runs a background goroutine that watches wall-clock silence on a running tool's output; past the timeout, it cancels a real `context.Context`, and `bash_exec` selects on that cancellation to call `killProcessGroup(cmd)` -- an actual `SIGKILL`-equivalent to the whole process group, so a hung shell command (and anything it spawned) is killed outright. The same context-cancellation path ends a judge's ephemeral sub-loop the instant `judge_verdict` settles, and backs Ctrl+C.

**The one genuinely soft layer** is the harness (`~/.kdeps/harness/*.yaml` -- tool-use rules, memory rules, honesty/scope/accuracy sections): prompt text asking the model to behave a certain way, with nothing in code checking whether it actually did. That's the dividing line for anything new: if it must be true regardless of what the model does, it's an event/action; if it's guidance the model is expected to follow, it's harness text.

## Enabling and disabling

Every event (and every harness section) can be turned off entirely, from the REPL, with the change persisted and picked up immediately -- no restart:

```
/harness events                          # list every event: trigger summary, action, enabled/disabled
/harness events disable identical-tool-calls
/harness events enable identical-tool-calls
```

This writes `disabled: true` (or removes it) in `~/.kdeps/events/<name>.yaml` -- the same override file `/konfig import` already writes to -- then reloads the registry in place. A disabled event's trigger can never fire: its threshold is treated as unreachably large (or, for `auto-compact`/`fold`, the loop skips the check outright, since their own context-window-aware branch would otherwise ignore an inflated threshold). Harness sections work the same way under the plain `/harness` command (`/harness list`, `/harness enable|disable <name>`) -- a disabled preamble section drops out of the system prompt on the very next turn, and a disabled standalone section (e.g. `m365-sandbox`) returns empty when looked up.

An unregistered or corrupted event/harness name always fails **open** (enabled) -- a missing config entry must never silently disable a safety mechanism; only an explicit `disabled: true` does.

## Occurrence caps on standalone harness text

Every corrective nudge the loop can send a model -- "make a real tool call instead of writing a fake result," "you said you can't, but tools just worked," and so on -- is a standalone harness entry (`nudge-action`, `nudge-fake-tool-response`, `nudge-sandbox-hallucination`, `nudge-unresolved-failure`, `nudge-give-up`), same as `tool-call-early-praise`'s positive-reinforcement text. These aren't sent unboundedly: an optional `maxOccurrences:` field caps how many times a given entry's *caller-tracked* counter may let it fire before the loop stops nudging and falls back to its non-nudge behavior (accepting the reply, flagging it, or failing the task):

```yaml
# ~/.kdeps/harness/nudge-give-up.yaml (built-in default -- shown for reference)
name: nudge-give-up
kind: standalone
maxOccurrences: 2   # push back on a premature "sorry, I can't" at most twice per turn
body: >-
  Your reply reads as giving up, but tool calls just succeeded this turn...
```

The count itself is tracked wherever it already lived before this existed -- a turn-scoped `turnNudges` field, or the session-scoped successful-tool-call counter behind `tool-call-early-praise` -- `maxOccurrences` only supplies the configurable ceiling that count is checked against. This is deliberately narrower than a generic "cadence" concept: a count-based cap fits naturally next to `disabled` on a standalone entry, but an *event-triggered* switch (like the sticky escalation from the short `tools-reminder` to the full `use-kdeps-tools` guidance once a sandbox hallucination has happened once this session) is a state transition, not a count, and stays as event/harness Go logic calling `harnessText`/`harnessRender` at the point the triggering event already fires -- the events system described above already owns "when does X happen," and duplicating that as a second cadence language on harness entries would just re-implement it worse.

0 (the default, omitted) means unlimited. `/harness list` shows a standalone entry's cap as `standalone, max Nx` when one is set.

## Presets

Hand-tuning fifteen different events one at a time to make a session cheaper (or more thorough) is tedious. A **preset** is a named, coherent bundle of event overrides -- applying one is like `/fold preset` but for the whole token-economy cluster at once, not just fold:

```
/harness preset                 # list built-in + user presets, with descriptions
/harness preset frugal          # apply: minimize token usage
/harness preset thorough        # apply: maximize context retention
/harness preset balanced        # apply: reset every tuned event back to its shipped default
```

Three presets ship today, all tuning the same fifteen events (`auto-compact`, `fold`, `memory-prompt-limit`, `memory-keys-limit`, `memory-focus-max`, `memory-chain-max`, `rel-memory-limit`, `tool-result-truncate`, `tool-error-truncate`, `force-answer-digest`, `history-window-trim`, `web-call-budget`, `bash-call-budget`, `file-call-budget`, `code-call-budget`) at three intensities:

| Preset | Use it for |
|---|---|
| `frugal` | cost-sensitive sessions or a small-context-window local model -- compacts and folds sooner, injects far less memory, truncates output harder, tighter call budgets |
| `balanced` | the shipped, out-of-the-box defaults -- also the "reset" preset after trying `frugal`/`thorough` |
| `thorough` | long research/investigation sessions where losing history hurts more than the extra tokens cost -- compacts/folds much less often, far more memory context, larger call budgets |

Applying a preset writes each tuned event straight to its normal `~/.kdeps/events/<name>.yaml` override file (exactly what `/harness events disable <name>` or a hand-edit would produce) and reloads the event registry immediately -- no restart, and no separate "preset" state to fall out of sync with the events it wrote. A preset never touches tuning/registry/theme/skills/actions, so switching between presets can never clobber an unrelated persisted setting.

Like harness sections and events, a preset is built-in-embed plus `~/.kdeps` user-override, merged by name: drop a `~/.kdeps/presets/frugal.yaml` to override the built-in `frugal`, or a differently-named file to add a preset of your own. A preset's shape is the same as its target override files -- an `events:` list of full event entries -- so authoring one is just collecting the event overrides you'd otherwise hand-write into one document with a `name:`/`description:`. Presets round-trip through [konfig](./konfig.md) export/import like every other registry.

## Status

Twenty-three events and ten actions ship today. Events: `auto-compact`, `fold`, `identical-tool-calls`, `convergence-block`, `task-round-budget`, `unproductive-rounds`, `handshake-timeout`, `judge-max-rounds`, `judge-iterations`, `tool-result-truncate`, `tool-error-truncate`, `force-answer-digest`, `history-window-trim`, `file-read-limit`, `web-call-budget`, `bash-call-budget`, `file-call-budget`, `code-call-budget`, `memory-prompt-limit`, `memory-keys-limit`, `memory-focus-max`, `memory-chain-max`, and `rel-memory-limit`. Every one can be listed and toggled via `/harness events` (see "Enabling and disabling" above); there is no `/event set <field> <value>` command yet for changing a trigger's numeric threshold from the REPL -- edit the YAML file directly for that, the same way a custom `/theme` or harness section is authored.

Not every hardcoded limit became an event: pure caps with no "measure, then fire one action" shape and no reasonable way to express as a bare `items:` count either -- the auto-generated judge panel's max size, per-turn nudge counts, log-line caps, a goroutine semaphore's buffer size -- stay plain Go constants. Turning every number in the codebase into an event would just move the same duplication into YAML instead of removing it.
