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

The four built-in call budgets are events whose `tools:` list names the tools they count. They fire on a *distinct-call count per turn*, so their `on:` clause uses `distinctCalls:`. A repeat call with the same arguments never counts twice and reuses the earlier answer.

```yaml
# ~/.kdeps/events/web-call-budget.yaml - replaces the shipped event outright
name: web-call-budget
tools: [web_search, wikipedia, web_scraper, exa_search, wolfram_alpha]  # add or remove a tool here
on:
  distinctCalls: 20   # DISTINCT calls to the tools above this turn -- they share one budget
run: block
```

| Event | Counts (shipped `tools:` list) | Default |
|---|---|---|
| `web-call-budget` | `web_search`, `wikipedia`, `web_scraper`, `exa_search`, `wolfram_alpha` | 20 |
| `bash-call-budget` | `bash_exec` commands | 50 |
| `file-call-budget` | `read_file` windows (path plus offset/limit) and `list_files` paths | 80 |
| `code-call-budget` | `search_local` queries | 30 |

A tool removed from a list runs with no budget and no reuse of earlier answers. A tool added to a list is counted by its arguments. These four events must keep a `tools:` list: an override file without one is rejected when it loads, and the shipped event stays in force.

These four already had a per-session override path before events existed -- `/model tool set web-limit <n>` (and `bash-limit`/`file-limit`/`code-limit`) still work exactly as before, on top of whatever the event's own default is, the same relationship `/fold threshold` has with the `fold` event. Setting one to `0` removes the cap entirely (`/model tool set web-limit 0`): the budget never blocks, and the system prompt stops telling the model to synthesize after a fixed number of searches - it states the limit actually enforced, or says there is no cap. Identical repeated calls (same call, same result) stop after as many rounds as the tool's allocated budget; only tools without a budget use the `identical-tool-calls` threshold.

## Your own tool budgets

An event with a `tools:` list is a budget you define in YAML, with no Go code that knows its name: in one turn the model may make at most `on.distinctCalls` distinct calls to those tools. A repeat of a call already made does not count again. Once the turn is over the limit, a new call is blocked the same way the built-in call budgets block one, and the model answers with what it has.

```yaml
# ~/.kdeps/events/search-budget.yaml - loaded at start, no registry entry needed
name: search-budget
tools: [web_search, web_scraper, wikipedia]   # the tools this budget counts together
on:
  distinctCalls: 6      # at most 6 distinct calls to these tools per turn
run: block              # the only action a tools: event takes
```

`/harness events disable search-budget` turns it off like any other event. A `tools:` event without `on.distinctCalls` or with any other `run:` is rejected when it loads.

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
| `memory-search-limit` | matches `memory_search` shows (best first; the rest are counted) | 20 |
| `memory-value-limit` (`on: bytes`) | bytes of one value `memory_search` shows, and of a summary section captured into memory | 500 |
| `memory-capture-limit` (`on: bytes`) | bytes of a tool result or thinking text captured into memory automatically | 300 |

```yaml
# ~/.kdeps/events/memory-value-limit.yaml
name: memory-value-limit
on:
  bytes: 1000   # memory_search shows up to 1000 bytes of each value, then "..."
run: truncate
```

## Thinking budgets

Each `/thinking` mode's reasoning token budget is an event: `thinking-budget-<mode>`, an `items:` count. The model may spend up to that many tokens reasoning before it answers; models that do not reason ignore it. A missing or disabled budget event uses 10000.

| Event | `/thinking` mode | Default |
|---|---|---|
| `thinking-budget-minimal` | `minimal` | 512 |
| `thinking-budget-low` | `low` | 2048 |
| `thinking-budget-medium` | `medium` | 8192 |
| `thinking-budget-high` | `high` | 16000 |
| `thinking-budget-xhigh` | `xhigh` | 32000 |
| `thinking-budget-auto` | `auto` (the default mode) | 10000 |

```yaml
# ~/.kdeps/events/thinking-budget-high.yaml
name: thinking-budget-high
items: 24000   # /thinking high lets the model reason for up to 24000 tokens
run: cap
```
