# Event presets

`/harness events` turns one event off; a preset writes a bundle of event overrides at once. The trigger list is on [Reactive LLM events](/agent/events).

## Enabling and disabling

Any event or harness section can be turned off from the REPL. The change persists and applies immediately, with no restart:

```
/harness events                          # list every event: trigger summary, action, enabled/disabled
/harness events disable identical-tool-calls
/harness events enable identical-tool-calls
```

- This writes (or removes) `disabled: true` in `~/.kdeps/events/<name>.yaml`, the file `/konfig import` writes, then reloads the registry.
- A disabled event never fires: its threshold becomes unreachable. For `auto-compact` and `fold`, the loop skips the check, since their context-window branch would ignore an inflated threshold.
- Harness sections use `/harness list` and `/harness enable|disable <name>`. A disabled preamble section leaves the system prompt on the next turn; a disabled standalone section (e.g. `m365-sandbox`) returns empty.
- An unknown or corrupted event or harness name fails open (enabled). Only an explicit `disabled: true` disables a safety mechanism.

## Occurrence caps on standalone harness text

Every corrective nudge is a standalone harness entry: `nudge-action`, `nudge-fake-tool-response`, `nudge-sandbox-hallucination`, `nudge-unbacked-claim`, `nudge-unresolved-failure`, `nudge-edit-unchanged`, `nudge-continue`, `nudge-self-limit`, and the praise text `tool-call-early-praise`. `maxOccurrences:` caps how often an entry fires. Past the cap, the loop falls back to accepting the reply, flagging it, or failing the task:

```yaml
# harness/nudge-self-limit.yaml (built-in default -- shown for reference)
name: nudge-self-limit
kind: standalone
maxOccurrences: 4   # push back on an invented limit ("I can't", "no tools") at most 4 times per turn
body: >-
  You have the kdeps tools in this turn, including the filesystem and the shell...
```

- The count is per turn for nudges, and per session for `tool-call-early-praise` (successful tool calls).
- `0` (the default) means unlimited, but only for a present entry with a body. A missing, disabled or empty entry never fires.
- `/harness list` shows a cap as `standalone, max Nx`.
- State switches are not caps. The sticky escalation from `tools-reminder` to the full `use-kdeps-tools` guidance after a sandbox hallucination is driven by [events](/agent/events).

## Presets

A preset is a named bundle of event overrides for the whole token-economy cluster, like `/fold preset` for every event at once:

```
/harness preset                 # list built-in + user presets, with descriptions
/harness preset frugal          # apply: minimize token usage
/harness preset thorough        # apply: maximize context retention
/harness preset balanced        # apply: reset every tuned event back to its shipped default
```

| Preset | Use it for |
|---|---|
| `frugal` | cost-sensitive sessions or small-context local models: compacts and folds sooner, less memory, harder truncation, tighter call budgets |
| `balanced` | the shipped defaults; also resets after `frugal` or `thorough` |
| `thorough` | long research sessions where lost history costs more than tokens: compacts and folds less, more memory, larger call budgets |
| `design-review` | reviewing a design or plan: sets a four-judge [panel](./judges.md#judge-panel-presets) and tunes no events |

`frugal`, `balanced` and `thorough` tune the same events: `auto-compact`, `fold`, `memory-prompt-limit`, `memory-keys-limit`, `memory-focus-max`, `memory-chain-max`, `rel-memory-limit`, `memory-search-limit`, `memory-value-limit`, `memory-capture-limit`, `tool-result-truncate`, `tool-error-truncate`, `force-answer-digest`, `history-window-trim`, `history-window-old-result`, `web-call-budget`, `bash-call-budget`, `file-call-budget`, `code-call-budget`, and the [efficiency](./efficiency.md) events.

Each preset requires the components whose tools its budget events limit: `web_search`, `web_scraper`, `bash_exec`, `exec`, `read_file`, `list_files`, `write_file`, `edit_file`, `search_local`, `memory_query`, `memory_search`. `kdeps update` installs them with the preset; each calls its tool through the kdeps API. Loop-wide events (compaction, truncation, history, stop rules) need no component.

## Applying a preset

A preset writes each tuned event to its `~/.kdeps/events/<name>.yaml` override (the same file `/harness events disable` writes) and reloads the registry. There is no separate preset state to drift. A preset never touches tuning, registry, theme, skills or actions settings.

## Authoring a preset

Presets merge built-ins with `~/.kdeps/presets/` by name. `~/.kdeps/presets/frugal.yaml` overrides `frugal`; another name adds a preset. A preset is a `name:`, a `description:` and an `events:` list of full event entries, the same shape as the override files. Presets round-trip through [konfig](./konfig.md) export/import.

## Status

Shipped events:

- **Efficiency** (11, see [efficiency enforcement](./efficiency.md)): `efficiency`, `efficiency-verbose`, `efficiency-reads`, `efficiency-actions`, `efficiency-stops`, `efficiency-failures`, `efficiency-tighten`, `efficiency-web`, `efficiency-bash`, `efficiency-file`, `efficiency-code`.
- **Other** (39): `auto-compact`, `fold`, `identical-tool-calls`, `convergence-block`, `handshake-timeout`, `judge-max-rounds`, `subagent-max-rounds`, `judge-iterations`, `tool-result-truncate`, `tool-error-truncate`, `force-answer-digest`, `history-window-trim`, `history-window-old-result`, `file-read-limit`, `web-call-budget`, `bash-call-budget`, `file-call-budget`, `code-call-budget`, `memory-prompt-limit`, `memory-keys-limit`, `memory-focus-max`, `memory-chain-max`, `rel-memory-limit`, `memory-search-limit`, `memory-value-limit`, `memory-capture-limit`, `memory-tool-results-limit`, `memory-thinking-limit`, `memory-low-signal-limit`, `memory-status-limit`, `model-record-limit`, `openrouter-free-per-minute`, `side-call-max-tokens`, and six `thinking-budget-<mode>` events.

Nine actions ship. `/harness events` lists and toggles every event. There is no REPL command for a trigger's numeric threshold; edit the YAML file.

Some limits stay Go constants: values with no "measure, then fire an action" shape and no `items:` count, such as the judge panel's max size, per-turn nudge counts, log-line caps and a semaphore's buffer size.
