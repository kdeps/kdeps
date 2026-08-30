# While-loop iteration

The [`loop`](/reference/glossary#loop) block enables conditional, unbounded iteration - making kdeps workflows Turing complete. It is a workflow mode concept, set on a resource; it also applies when the LLM calls a workflow as a tool in agent mode. Unlike `items` (which iterates over a fixed list), `loop` repeats a resource body while an optional expression is true (or for a fixed count when `while:` is omitted), with full access to mutable state via `set()`/`get()`. Add `every:` to turn the loop into a **repeated scheduled task** that pauses for a fixed duration between iterations.

## Basic usage

<div v-pre>

```yaml
# resources/count.yaml

actionId: countToFive
name: Count to Five
loop:
  while: "loop.index() < 5"
  maxIterations: 1000   # safety cap (default: 1000)
after:
  - "{{ set('result', loop.count()) }}"
apiResponse:
  success: true
  response:
    count: "{{ get('result') }}"
```

</div>

`loop.index() < 5` runs the body for index 0 - 4, producing 5 iterations. Each iteration's `apiResponse` becomes one element of the streaming response.

## Loop context

Inside the loop body, special callables are available:

| Callable | Description |
|----------|-------------|
| `loop.index()` | Current iteration index (0-based) |
| `loop.count()` | Current iteration count (1-based) |
| `loop.results()` | Results accumulated from **all prior** iterations |

### Method syntax

```yaml
# resources/example.yaml
after:
  - "{{ set('idx', loop.index()) }}"
  - "{{ set('cnt', loop.count()) }}"
  - "{{ set('prev', loop.results()) }}"
```

### Comparison: `loop` vs `item`

| Loop | Items equivalent | Description |
|------|-----------------|-------------|
| `loop.index()` | `item.index()` | Current index (0-based) |
| `loop.count()` | `item.count()` | Current count (1-based) |
| `loop.results()` | `item.values()` | All prior results |
| `set('key', val, 'loop')` | `set('key', val, 'item')` | Loop-scoped storage |
| `get('key', 'loop')` | `get('key', 'item')` | Read loop-scoped value |

## Loop-scoped storage

Use `'loop'` as a storage type hint to scope variables to the loop context, mirroring the `'item'` type for items iteration:

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "default(get('step', 'loop'), 0) < 3"
  maxIterations: 10
after:
  - "{{ set('step', loop.count(), 'loop') }}"
```

</div>

## `loop.results()` - self-referential termination

`loop.results()` returns a slice of all results from **previous** iterations. This enables patterns where the termination condition depends on what the loop has already produced:

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "len(loop.results()) < 3"
  maxIterations: 10
after:
  - "{{ set('n', loop.count()) }}"
```

</div>

The loop runs until 3 results have been collected, regardless of how many iterations that takes - a key pattern for mu-recursion and unbounded search.

## Streaming response

When `apiResponse` is present, every iteration produces one response map. Multiple per-iteration responses constitute a **streaming response** - a slice returned to the caller. This mirrors how `items` with `apiResponse` works.

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "loop.index() < 3"
  maxIterations: 10
after:
  - "{{ set('tick', loop.count()) }}"
apiResponse:
  success: true
  response:
    tick: "{{ get('tick') }}"
```

</div>

Three iterations → three `apiResponse` maps → streaming slice of length 3.

No iterations → empty slice.

## `maxIterations` safety cap

`maxIterations` is a configurable upper bound on the number of iterations. It prevents accidental infinite loops in production while preserving Turing completeness - users can set it to any positive integer.

- Default: `1000`
- Set to any positive integer for tighter or looser control
- Turing completeness is preserved because the cap is configurable, not fixed

```yaml
# resources/example.yaml
loop:
  while: "true"
  maxIterations: 50000   # allow up to 50k iterations
```

## `every:` - repeated scheduled tasks

Add `every:` to pause the loop for a fixed duration **between** iterations, turning it into a repeated scheduled task (ticker pattern). Supported units: `ms` (milliseconds), `s` (seconds), `m` (minutes), `h` (hours).

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "loop.index() < 10"
  every: "5s"           # wait 5 seconds between each iteration
  maxIterations: 100
after:
  - "{{ set('tick', loop.count()) }}"
apiResponse:
  success: true
  response:
    tick: "{{ get('tick') }}"
    at:   "{{ loop.count() }}"
```

</div>

The sleep is **skipped after the last iteration** - the caller receives results without an unnecessary trailing delay.

### Combining `while: "true"` with `every:` for infinite polling

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "true"          # run until maxIterations
  every: "30s"           # poll every 30 seconds
  maxIterations: 1440    # up to 12 hours (1440 × 30 s)
exec:
  command: "poll-service.sh"
after:
  - "{{ get('execResource').exitCode == 0 ? set('done', true) : set('noop', 0) }}"
```

</div>

### Duration format

| Example | Meaning |
|---------|---------|
| `"500ms"` | 500 milliseconds |
| `"1s"` | 1 second |
| `"2m"` | 2 minutes |
| `"1h"` | 1 hour |

An invalid `every:` value (e.g. `"not-a-duration"`) is rejected at validation time.

## `at:` - specific dates and times

Use `at:` to fire the loop body at a list of specific dates and/or times, in order. The engine sleeps until each scheduled time before executing the body for that iteration.

`every:` and `at:` are mutually exclusive - set only one per loop block.

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "loop.index() < 2"
  maxIterations: 10
  at:
    - "2026-03-15T10:00:00Z"   # RFC3339 absolute timestamp
    - "2026-03-15T14:30:00Z"
after:
  - "{{ set('tick', loop.count()) }}"
apiResponse:
  success: true
  response:
    tick: "{{ get('tick') }}"
```

</div>

### Supported date/time formats

| Format | Example | Behaviour |
|--------|---------|-----------|
| RFC3339 (UTC) | `"2026-03-15T10:00:00Z"` | Fire at that exact instant |
| RFC3339 (offset) | `"2026-03-15T10:00:00+02:00"` | Fire at that exact instant |
| Local datetime | `"2026-03-15T10:00:00"` | Treated as local time |
| Time of day | `"10:00"` or `"10:00:00"` | Next occurrence of that time today; tomorrow if already past |
| Date only | `"2026-03-15"` | Midnight (00:00:00) of that date, local time |

### Daily recurring example

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "true"
  maxIterations: 30  # run for 30 days
  at:
    - "08:00"   # fire at 08:00 every morning (next occurrence)
    - "20:00"   # fire at 20:00 every evening
exec:
  command: "daily-report.sh"
```

</div>

If a time entry is already in the past the engine fires immediately (no sleep).  
An invalid entry (e.g. `"not-a-date"`) causes an error before any iterations run.

## Condition syntax

The `while:` field is optional. When omitted, the loop runs until `maxIterations` (default 1000) or until all `at:` entries are consumed.

When provided, the expression is evaluated using expr-lang - any boolean expression is valid:

```yaml
# Counter
while: "loop.index() < 10"

# Data-driven termination
while: "get('done') == nil"

# Prior-results driven
while: "len(loop.results()) < 5"

# Mutable state
while: "int(default(get('phase'), 0)) < 3"

# Mathematical search (mu-recursion)
while: "int(loop.count()) * int(loop.count() + 1) / 2 <= 20"
```

The `while` field is a plain expr-lang boolean expression string.

## Turing completeness

The three primitives of Turing completeness are:

| Primitive | How kdeps provides it |
|-----------|----------------------|
| **Unbounded iteration** | `loop.while` with configurable `maxIterations` |
| **Mutable state** | `set()` / `get()` across iterations |
| **Conditional branching** | Arbitrary boolean `while` expression + `validations.skip` |

Together with `loop.results()` feeding back into the `while` condition, the system can simulate any computable function - including mu-recursion (search until an unpredictable condition is met).

## Examples

### Accumulator (sum 1+2+3+4 = 10)

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "loop.index() < 4"
  maxIterations: 100
after:
  - "{{ set('sum', int(default(get('sum'), 0)) + loop.count()) }}"
apiResponse:
  success: true
  response:
    partial_sum: "{{ get('sum') }}"
```

</div>

### State-machine phase transition

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "int(default(get('phase'), 0)) < 3"
  maxIterations: 10
after:
  - "{{ set('phase', int(default(get('phase'), 0)) + 1) }}"
apiResponse:
  success: true
  response:
    phase: "{{ get('phase') }}"
```

</div>

### Conditional early exit (flag-based)

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "get('done') == nil"
  maxIterations: 100
exec:
  command: "check-condition.sh"
after:
  - "{{ get('execResource').exitCode == 0 ? set('done', true) : set('noop', 0) }}"
apiResponse:
  success: true
  response:
    iterations: "{{ loop.count() }}"
```

</div>

### Collect n results

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "len(loop.results()) < 5"
  maxIterations: 50
chat:
  prompt: "Generate item {{ loop.count() }}"
apiResponse:
  success: true
  response:
    item: "{{ get('chatResource') }}"
```

</div>

### Downstream resource reads loop output

A resource that runs a loop and a downstream resource that reads the final state:

<div v-pre>

```yaml
# resources/compute.yaml
actionId: compute
loop:
  while: "loop.index() < 3"
  maxIterations: 10
after:
  - "{{ set('computed', loop.count()) }}"

---
# resources/respond.yaml
actionId: respond
requires: [compute]
apiResponse:
  success: true
  response:
    value: "{{ get('computed') }}"   # reads final value set by the loop
```

</div>

## When to use `loop` vs `items`

| Use `loop` when... | Use `items` when... |
|------------------|------------------|
| Number of iterations is not known in advance | You have a fixed list to process |
| Termination depends on runtime state | You want to iterate over a pre-computed array |
| You need mutable accumulation across iterations | Each item is independent |
| Implementing search / retry / polling patterns | Batch processing of a dataset |
| Running a repeated scheduled task (`every:`) | One-shot batch over a dataset |
| Firing at specific dates or times (`at:`) | Batch processing of a dataset |

## See also

- [Items Iteration](items) - Fixed-list iteration
- [Expressions](/concepts/expressions) - Expression syntax
- [Expression Functions Reference](/reference/expression-functions-reference) - Complete function list
- [Validation and Control Flow](/concepts/validation-and-control) - Skip conditions, preflight checks
