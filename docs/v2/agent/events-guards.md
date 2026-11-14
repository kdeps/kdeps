# Event guards

These events fire on a round count, a truncation size, a call budget, or a memory cap. The two token events, auto-compact and fold, stay on [Reactive LLM events](/agent/events).

## Round-count events

The same `{name, on, run}` shape covers the stuck-loop and task-budget guards. They fire on a count of rounds, not tokens, so `on:` uses `rounds:`:

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
| `agent-max-rounds` | an agent-loop turn spends this many tool rounds (200) when `/model tool set rounds` sets none | forces a text answer |
| `chat-max-rounds` | a workflow chat with no `tools.maxRounds` spends this many tool rounds (5) | forces a text answer |
| `subagent-max-rounds` | one [subagent](./subagent.md) task spends this many tool rounds (20) | ends the subagent and forces its report |
| `judge-iterations` | the revise-and-rejudge loop retries this many times after a judge rejection | accepts the last response regardless of outcome -- a judge panel must never block a turn indefinitely |

## A sensible floor of 1

Each has a built-in floor of 1. An override with no `rounds:`, or a corrupted override, never becomes 0. A value of 0 starts the event on the first round.

## Truncation events

The same shape covers the loop's token-economy limits -- these fire on a *measured byte length*, so their `on:` clause uses `bytes:` instead:

```yaml
# ~/.kdeps/events/tool-result-truncate.yaml
name: tool-result-truncate
on:
  bytes: 8192   # any single tool result past this size keeps its start and end
run: truncate
```

| Event | Fires when | Action |
|---|---|---|
| `tool-result-truncate` | a single tool result fed back to the LLM exceeds this size | keeps its start and end on line boundaries, marks the cut |
| `tool-error-truncate` | a tool's failure text exceeds this size | truncates before display and before feeding it back to the LLM |
| `force-answer-digest` | the gathered-output digest inlined into a forced answer exceeds this size | truncates the digest |
| `history-window-trim` | the in-flight tool-loop message array exceeds this size | drops the oldest complete tool round-trips |
| `history-window-old-result` | a tool result older than the two most recent steps of the turn exceeds this size | cuts it to its start, notes how much was dropped |
| `file-read-limit` | a file a builtin tool is about to read (or `write_file`'s own content) exceeds this size | rejects the call with an error instead of reading/writing it |

Each also falls back to its built-in default if the registry is unavailable. An override that omits `bytes:` never means "truncate to nothing" or "reject every file".

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

If you remove a tool from a list, the tool runs with no budget and does not use earlier answers again. If you add a tool to a list, kdeps counts it by its arguments. These four events must keep a `tools:` list. kdeps rejects an override file without this list when it loads. The shipped event then stays active.

`/model tool set web-limit <n>` (and `bash-limit`, `file-limit`, `code-limit`) overrides an event's default for the session, as `/fold threshold` does for `fold`. `0` removes the cap (`/model tool set web-limit 0`): the budget never blocks, and the system prompt states the enforced limit or that there is none. Identical repeated calls (same call, same result) stop after as many rounds as the tool's budget. Tools without a budget use the `identical-tool-calls` threshold.

Workflow chats read the same events: each chat call counts its own distinct calls, and a repeated call reuses its first result (see [tool loop behavior](/workflow/resources/llm#tool-loop-behavior)).

## Your own tool budgets

An event with a `tools:` list is a budget defined only in YAML. In one turn, the model can make a maximum of `on.distinctCalls` different calls to those tools. A repeated call does not count again. After the limit, kdeps blocks a new call, as for the built-in budgets. The model then answers with the data that it has.

```yaml
# ~/.kdeps/events/search-budget.yaml - loaded at start, no registry entry needed
name: search-budget
tools: [web_search, web_scraper, wikipedia]   # the tools this budget counts together
on:
  distinctCalls: 6      # at most 6 distinct calls to these tools per turn
run: block              # the only action a tools: event takes
```

`/harness events disable search-budget` turns it off like any other event. kdeps rejects a `tools:` event without `on.distinctCalls` or with a different `run:` when it loads.

## Memory events

The [persistent memory](./memory.md) injection and traversal caps are events too. `memory-prompt-limit` fires on a token measurement like `auto-compact` and `fold`. The rest are plain counts that always apply, so they set `items:` instead of `on:`:

```yaml
# ~/.kdeps/events/memory-focus-max.yaml
name: memory-focus-max
items: 5   # how many prompt-relevant entries are force-kept when memory is truncated
run: cap
```

| Event | Caps | Default |
|---|---|---|
| `memory-prompt-limit` | tokens of memory content injected into the system preamble each turn | 500 |
| `memory-keys-limit` | key names listed in the `<memory-keys>` preamble block | 25 |
| `memory-focus-max` | prompt-relevant entries force-kept when memory is truncated | 5 |
| `memory-chain-max` | entries the active task chain force-keeps (its nearest ancestors) | 8 |
| `rel-memory-limit` | base memory rows fed into a `memory_query` join, bounding its worst case | 500 |
| `memory-search-limit` | matches `memory_search` shows (best first; the rest are counted) | 20 |
| `memory-value-limit` (`on: bytes`) | bytes of one value `memory_search` shows, and of a summary section captured into memory | 500 |
| `memory-capture-limit` (`on: bytes`) | bytes of a tool result or thinking text captured into memory automatically | 300 |
| `memory-tool-results-limit` | automatic captures kept per tool (oldest dropped) | 20 |
| `memory-thinking-limit` | reasoning (thinking) entries kept (oldest dropped) | 20 |
| `memory-low-signal-limit` | entries of the low-signal types (`low_signal:` in [memory rules](/agent/memory-internals#memory-rules)) kept (oldest dropped) | 50 |
| `memory-status-limit` | tool-call status entries (`status:<tool>:<n>`) kept (oldest dropped) | 60 |
| `memory-key-chars` (`on: bytes`) | characters of a memory key, and of a memory note's file name made from it | 80 |
| `memory-tool-line-chars` (`on: bytes`) | characters of a tool result's first line used to name its entry | 40 |
| `memory-derived-key-chars` (`on: bytes`) | characters of a key made from a tool name and its result | 30 |
| `memory-summary-chars` (`on: bytes`) | characters of a memory note's summary line (longer is cut with `...`) | 120 |
| `memory-focus-min-word` | the shortest prompt word that counts when the prompt is matched to memory | 3 |
| `memory-format-tokens` | tokens of memory put in a prompt when the caller sets no budget | 2000 |
| `memory-query-rows-max` | rows `memory_query` returns for a relation, for all `limit` values (keeps the result under the tool result cap) | 500 |

Two agent loop budgets are events of the same kind, read only when the config leaves them unset:

| Event | What it caps | Default |
|---|---|---|
| `tool-allocation-increment` | tool rounds added each time auto tool allocation raises the budget | 100 |
| `preamble-token-limit` | tokens of the system preamble kept when neither the compact token budget nor the auto-compact threshold is set | 40000 |

```yaml
# ~/.kdeps/events/memory-value-limit.yaml
name: memory-value-limit
on:
  bytes: 1000   # memory_search shows up to 1000 bytes of each value, then "..."
run: truncate
```

## Thinking budgets

Each `/thinking` mode's reasoning token budget is an event: `thinking-budget-<mode>`, an `items:` count. The model can use a maximum of that number of tokens to reason before it answers. Models that do not reason ignore it. A workflow chat's `thinking:` mode with no `budgetTokens` uses the same event. A missing or disabled budget event uses 10000.

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
