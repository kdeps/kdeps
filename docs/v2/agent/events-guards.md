# Event guards

These events fire on a round count, a truncation size, a call budget, or a memory cap. The two token events, auto-compact and fold, stay on [Reactive LLM events](/agent/events).

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
| `identical-tool-calls` | the model issues the exact same tool call and gets the exact same result this many times in a row. Tools with a per-turn budget (web, bash, file, code) ignore this number and use their allocated budget instead (`/model tool set <category>-limit <n>`; `0` = unlimited = never stops) | ends the turn as a stuck loop |
| `convergence-block` | this many consecutive rounds end in a convergence-blocked tool call (the model kept trying once a web/bash/file/code budget ran out) | strips every tool and forces a text answer |
| `task-round-budget` | a single goal-directed task spends this many tool rounds | force-closes the task |
| `unproductive-rounds` | this many consecutive rounds produce no new tool result or state change | force-closes the task and fails it forward |
| `handshake-timeout` | a session-handshake grounding challenge (see [session-integrity handshake](/agent/tools-handshake#session-integrity-handshake)) spends this many tool rounds without resolving | fails the handshake |
| `judge-max-rounds` | a single [judge](./judges.md)'s review spends this many tool rounds | ends that judge's review and forces its verdict |
| `judge-iterations` | the revise-and-rejudge loop retries this many times after a judge rejection | accepts the last response regardless of outcome -- a judge panel must never block a turn indefinitely |

## A sensible floor of 1

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

These four already had a per-session override path before events existed -- `/model tool set web-limit <n>` (and `bash-limit`/`file-limit`/`code-limit`) still work exactly as before, on top of whatever the event's own default is, the same relationship `/fold threshold` has with the `fold` event. Setting one to `0` removes the cap entirely (`/model tool set web-limit 0`): the budget never blocks, and the system prompt stops telling the model to synthesize after a fixed number of searches - it states the limit actually enforced, or says there is no cap. Identical repeated calls (same call, same result) stop after as many rounds as the tool's allocated budget; only tools without a budget use the `identical-tool-calls` threshold.

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
