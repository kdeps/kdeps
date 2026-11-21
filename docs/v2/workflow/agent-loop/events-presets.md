# Event presets

An override file turns off one event. A preset is a named set of event overrides. The trigger list is on [Reactive LLM events](/workflow/agent-loop/events).

## Enabling and disabling

Set `disabled: true` in an override file. kdeps reads it on the next run:

```yaml
# ~/.kdeps/events/identical-tool-calls.yaml (or events/ in the workflow project)
name: identical-tool-calls
disabled: true
```

- A disabled event never starts, because its threshold becomes too high to reach. For `auto-compact` and `fold`, the loop skips the check. Their context-window branch ignores a high threshold, so the check is necessary.
- A harness section takes the same `disabled: true` in `~/.kdeps/harness/<name>.yaml`. A disabled preamble section leaves the system prompt. A disabled standalone section (for example `tools-reminder`) returns empty.
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
- State switches are not caps. After a sandbox hallucination, `tools-reminder` changes permanently to the full `use-kdeps-tools` guidance. [Events](/workflow/agent-loop/events) control this change.

## Presets

A preset is a named set of event values for the full token-economy cluster:

| Preset | Use it for |
|---|---|
| `frugal` | cost-sensitive sessions or small-context local models: compacts and folds sooner, less memory, harder truncation, tighter call budgets |
| `balanced` | the shipped defaults; also resets after `frugal` or `thorough` |
| `thorough` | long research sessions where lost history costs more than tokens: compacts and folds less, more memory, larger call budgets |
| `design-review` | stores a four-judge panel a workflow chat names with `judgePreset`. It tunes no events. |

`frugal`, `balanced` and `thorough` tune the same events: `auto-compact`, `fold`, `memory-prompt-limit`, `memory-keys-limit`, `memory-focus-max`, `memory-chain-max`, `rel-memory-limit`, `memory-search-limit`, `memory-value-limit`, `memory-capture-limit`, `tool-result-truncate`, `tool-error-truncate`, `force-answer-digest`, `history-window-trim`, `history-window-old-result`, `web-call-budget`, `bash-call-budget`, `file-call-budget`, `code-call-budget`.

Each preset requires the components whose tools its budget events limit: `web_search`, `web_scraper`, `bash_exec`, `exec`, `read_file`, `list_files`, `write_file`, `edit_file`, `search_local`, `memory_query`, `memory_search`. `kdeps update` installs them with the preset. Each calls its tool through the kdeps API. Loop-wide events (compaction, truncation, history, stop rules) need no component.

## Using a preset

A chat uses a preset's judge panel with `judgePreset:` (see [Judges](/workflow/resources/llm#judges)). To use a preset's event values, copy its `events:` entries into override files in `~/.kdeps/events/` or the workflow's `events/` folder. A workflow folder wins over `~/.kdeps` (see [Answer quality](/workflow/quality)).

## Authoring a preset

Presets merge built-ins with `~/.kdeps/presets/` by name. `~/.kdeps/presets/frugal.yaml` overrides `frugal`. Another name adds a preset. A preset is a `name:`, a `description:` and an `events:` list of full event entries, the same shape as the override files.

## Status

Shipped events are the files in `assets/events`.

Eight actions ship. To change a threshold, edit the event's YAML file.

Some limits stay Go constants. These values do not measure and then start an action, and they have no `items:` count. Examples are the maximum size of the judge panel, the nudge count for each turn, the log-line limits, and the buffer size of a semaphore.
