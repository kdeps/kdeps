# Event presets

`/harness events` turns one event off. A preset writes a whole bundle of those overrides at once. The trigger list stays on [Reactive LLM events](/agent/events).

## Enabling and disabling

Every event (and every harness section) can be turned off entirely, from the REPL, with the change persisted and picked up immediately -- no restart:

```
/harness events                          # list every event: trigger summary, action, enabled/disabled
/harness events disable identical-tool-calls
/harness events enable identical-tool-calls
```

This writes `disabled: true` (or removes it) in `~/.kdeps/events/<name>.yaml` -- the same override file `/konfig import` already writes to -- then reloads the registry in place. A disabled event's trigger can never fire: its threshold is treated as unreachably large (or, for `auto-compact`/`fold`, the loop skips the check outright, since their own context-window-aware branch would otherwise ignore an inflated threshold). Harness sections work the same way under the plain `/harness` command (`/harness list`, `/harness enable|disable <name>`) -- a disabled preamble section drops out of the system prompt on the very next turn, and a disabled standalone section (e.g. `m365-sandbox`) returns empty when looked up.

An unregistered or corrupted event/harness name always fails open (enabled) -- a missing config entry must never silently disable a safety mechanism; only an explicit `disabled: true` does.

## Occurrence caps on standalone harness text

Every corrective nudge the loop can send a model -- "make a real tool call instead of writing a fake result," "you have the tools, call one," and so on -- is a standalone harness entry (`nudge-action`, `nudge-fake-tool-response`, `nudge-sandbox-hallucination`, `nudge-unbacked-claim`, `nudge-unresolved-failure`, `nudge-edit-unchanged`, `nudge-continue`, `nudge-self-limit`), same as `tool-call-early-praise`'s positive-reinforcement text. These are not sent unboundedly: an optional `maxOccurrences:` field caps how many times a given entry's *caller-tracked* counter may let it fire before the loop stops nudging and falls back to its non-nudge behavior (accepting the reply, flagging it, or failing the task):

```yaml
# harness/nudge-self-limit.yaml (built-in default -- shown for reference)
name: nudge-self-limit
kind: standalone
maxOccurrences: 4   # push back on an invented limit ("I can't", "no tools") at most 4 times per turn
body: >-
  You have the kdeps tools in this turn, including the filesystem and the shell...
```

The count itself is tracked wherever it already lived before this existed -- a turn-scoped `turnNudges` field, or the session-scoped successful-tool-call counter behind `tool-call-early-praise` -- `maxOccurrences` only supplies the configurable ceiling that count is checked against. This is deliberately narrower than a generic "cadence" concept: a count-based cap fits naturally next to `disabled` on a standalone entry, but an *event-triggered* switch (like the sticky escalation from the short `tools-reminder` to the full `use-kdeps-tools` guidance once a sandbox hallucination has happened once this session) is a state transition, not a count, and stays as event/harness Go logic calling `harnessText`/`harnessRender` at the point the triggering event already fires -- the [events](/agent/events) system already owns "when does X happen," and duplicating that as a second cadence language on harness entries would just re-implement it worse.

0 (the default, omitted) means unlimited, but only for an entry that is present and has a body. A missing entry, a disabled entry, or an entry whose body is empty does not fire. It is not treated as unlimited: a nudge whose YAML is not in the loaded asset set must not keep appending the invoke example. `/harness list` shows a standalone entry's cap as `standalone, max Nx` when one is set.

## Presets

Hand-tuning dozens of events one at a time to make a session cheaper (or more thorough) is tedious. A **preset** is a named, coherent bundle of event overrides -- applying one is like `/fold preset` but for the whole token-economy cluster at once, not just fold:

```
/harness preset                 # list built-in + user presets, with descriptions
/harness preset frugal          # apply: minimize token usage
/harness preset thorough        # apply: maximize context retention
/harness preset balanced        # apply: reset every tuned event back to its shipped default
```

Three presets ship today, all tuning the same events (`auto-compact`, `fold`, `memory-prompt-limit`, `memory-keys-limit`, `memory-focus-max`, `memory-chain-max`, `rel-memory-limit`, `memory-search-limit`, `memory-value-limit`, `memory-capture-limit`, `tool-result-truncate`, `tool-error-truncate`, `force-answer-digest`, `history-window-trim`, `web-call-budget`, `bash-call-budget`, `file-call-budget`, `code-call-budget`, plus the ten [efficiency](./efficiency.md) events) at three intensities:

Each preset also requires the components whose tools its budget events limit: `web_search`, `web_scraper`, `bash_exec`, `exec`, `read_file`, `list_files`, `write_file`, `edit_file`, `search_local`, `memory_query`, `memory_search`. `kdeps update` installs them with the preset, and each one calls its tool through the kdeps API. Events that act on the loop as a whole (compaction, truncation, history, stop rules) require no component.

| Preset | Use it for |
|---|---|
| `frugal` | cost-sensitive sessions or a small-context-window local model -- compacts and folds sooner, injects far less memory, truncates output harder, tighter call budgets |
| `balanced` | the shipped, out-of-the-box defaults -- also the "reset" preset after trying `frugal`/`thorough` |
| `thorough` | long research/investigation sessions where losing history hurts more than the extra tokens cost -- compacts/folds much less often, far more memory context, larger call budgets |

## Applying a preset

Applying a preset writes each tuned event straight to its normal `~/.kdeps/events/<name>.yaml` override file (exactly what `/harness events disable <name>` or a hand-edit would produce) and reloads the event registry immediately -- no restart, and no separate "preset" state to fall out of sync with the events it wrote. A preset never touches tuning/registry/theme/skills/actions, so switching between presets can never clobber an unrelated persisted setting.

## Authoring a preset

Like harness sections and events, a preset is built-in-embed plus `~/.kdeps` user-override, merged by name: drop a `~/.kdeps/presets/frugal.yaml` to override the built-in `frugal`, or a differently-named file to add a preset of your own. A preset's shape is the same as its target override files -- an `events:` list of full event entries -- so authoring one is just collecting the event overrides you would otherwise hand-write into one document with a `name:`/`description:`. Presets round-trip through [konfig](./konfig.md) export/import like every other registry.

## Status

Eleven more events (`efficiency`, `efficiency-verbose`, `efficiency-reads`, `efficiency-actions`, `efficiency-stops`, `efficiency-failures`, `efficiency-tighten`, `efficiency-web`, `efficiency-bash`, `efficiency-file`, `efficiency-code`) drive [efficiency enforcement](./efficiency.md). Thirty-six other events and nine actions ship today. Events: `auto-compact`, `fold`, `identical-tool-calls`, `convergence-block`, `handshake-timeout`, `judge-max-rounds`, `judge-iterations`, `tool-result-truncate`, `tool-error-truncate`, `force-answer-digest`, `history-window-trim`, `file-read-limit`, `web-call-budget`, `bash-call-budget`, `file-call-budget`, `code-call-budget`, `memory-prompt-limit`, `memory-keys-limit`, `memory-focus-max`, `memory-chain-max`, `rel-memory-limit`, `memory-search-limit`, `memory-value-limit`, `memory-capture-limit`, `memory-tool-results-limit`, `memory-thinking-limit`, `memory-low-signal-limit`, `memory-status-limit`, `model-record-limit`, `side-call-max-tokens`, and the six `thinking-budget-<mode>` events. Every one can be listed and toggled via `/harness events` (see "Enabling and disabling" above); there is no `/event set <field> <value>` command yet for changing a trigger's numeric threshold from the REPL -- edit the YAML file directly for that, the same way a custom `/theme` or harness section is authored.

Not every hardcoded limit became an event: pure caps with no "measure, then fire one action" shape and no reasonable way to express as a bare `items:` count either -- the auto-generated judge panel's max size, per-turn nudge counts, log-line caps, a goroutine semaphore's buffer size -- stay plain Go constants. Turning every number in the codebase into an event would just move the same duplication into YAML instead of removing it.
