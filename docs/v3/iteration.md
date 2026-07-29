# Iteration

Two loops: **`items:`** (for-each) and **`loop:`** (while). Both work in workflow mode; agent mode runs them inside a called workflow tool.

## items: (for-each)

```yaml
actionId: classifyMessages
items:
  - "Meeting at 3pm"
  - "Click here to claim prize"
chat:
  model: llama3.2:1b
  prompt: "Spam or not: {{ get('current') }}"
  jsonResponse: true
```

Dynamic list from a prior step:

```yaml
requires: [fetchUrls]
items: "{{ get('fetchUrls') }}"
scraper:
  url: "{{ get('current').url }}"
```

### Per-item getters

| Getter | Meaning |
|--------|---------|
| `get('current')` | Current item |
| `get('prev')` / `get('next')` | Neighbors (null at ends) |
| `get('index')` | 0-based index |
| `get('count')` | Length |
| `get('all')` | Full list |

Or `item.current()`, `item.index()`, `item.count()`, …

### Item scope

```yaml
before:
  - set('running_total', (get('running_total', 'item') or 0) + get('current').amount, 'item')
```

Item-scoped values do not leak to the parent store. After the resource finishes, `get('actionId')` is an **array** of iteration outputs.

## loop: (while)

```yaml
actionId: pollJob
loop:
  while: get('status') != 'done'
  maxIterations: 30
httpClient:
  url: "https://api.example.com/jobs/{{ get('jobId') }}"
after:
  - set('status', get('pollJob').status)
```

Use for poll-until-ready, search-until-found, short chain-of-thought cycles. Always set `maxIterations`.

## Choose

| Need | Use |
|------|-----|
| Fixed or known list | `items:` |
| Condition / poll | `loop:` |
| Multi-agent steps | [Agencies](/agencies) + `agent:` |

[Expressions](/expressions) · [Errors](/errors) · [Resources](/resources).
