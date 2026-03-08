# While-Loop Iteration

The `loop` block enables conditional, unbounded iteration — making kdeps workflows Turing complete. Unlike `items` (which iterates over a fixed list), `loop` repeats a resource body while an arbitrary expression is true, with full access to mutable state via `set()`/`get()`.

## Basic Usage

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: countToFive
  name: Count to Five

run:
  loop:
    while: "loop.index() < 5"
    maxIterations: 1000   # safety cap (default: 1000)
  expr:
    - "{{ set('result', loop.count()) }}"
  apiResponse:
    success: true
    response:
      count: "{{ get('result') }}"
```

</div>

`loop.index() < 5` runs the body for index 0–4, producing 5 iterations. Each iteration's `apiResponse` becomes one element of the streaming response.

## Loop Context

Inside the loop body, special callables are available:

| Callable | Description |
|----------|-------------|
| `loop.index()` | Current iteration index (0-based) |
| `loop.count()` | Current iteration count (1-based) |
| `loop.results()` | Results accumulated from **all prior** iterations |

### Method Syntax

```yaml
run:
  expr:
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

## Loop-Scoped Storage

Use `'loop'` as a storage type hint to scope variables to the loop context, mirroring the `'item'` type for items iteration:

<div v-pre>

```yaml
run:
  loop:
    while: "default(get('step', 'loop'), 0) < 3"
    maxIterations: 10
  expr:
    - "{{ set('step', loop.count(), 'loop') }}"
```

</div>

## `loop.results()` — Self-Referential Termination

`loop.results()` returns a slice of all results from **previous** iterations. This enables patterns where the termination condition depends on what the loop has already produced:

<div v-pre>

```yaml
run:
  loop:
    while: "len(loop.results()) < 3"
    maxIterations: 10
  expr:
    - "{{ set('n', loop.count()) }}"
```

</div>

The loop runs until 3 results have been collected, regardless of how many iterations that takes — a key pattern for mu-recursion and unbounded search.

## Streaming Response

When `apiResponse` is present, every iteration produces one response map. Multiple per-iteration responses constitute a **streaming response** — a slice returned to the caller. This mirrors how `items` with `apiResponse` works.

<div v-pre>

```yaml
run:
  loop:
    while: "loop.index() < 3"
    maxIterations: 10
  expr:
    - "{{ set('tick', loop.count()) }}"
  apiResponse:
    success: true
    response:
      tick: "{{ get('tick') }}"
```

</div>

Three iterations → three `apiResponse` maps → streaming slice of length 3.

No iterations → empty slice.

## `maxIterations` Safety Cap

`maxIterations` is a configurable upper bound on the number of iterations. It prevents accidental infinite loops in production while preserving Turing completeness — users can set it to any positive integer.

- Default: `1000`
- Set to any positive integer for tighter or looser control
- Turing completeness is preserved because the cap is configurable, not fixed

```yaml
run:
  loop:
    while: "true"
    maxIterations: 50000   # allow up to 50k iterations
```

## Condition Syntax

The `while` expression is evaluated using expr-lang. Any boolean expression is valid:

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

The `while` field optionally accepts Mustache wrappers (<span v-pre>`{{ }}`</span>); they are stripped automatically.

## Turing Completeness

The three primitives of Turing completeness are:

| Primitive | How kdeps provides it |
|-----------|----------------------|
| **Unbounded iteration** | `loop.while` with configurable `maxIterations` |
| **Mutable state** | `set()` / `get()` across iterations |
| **Conditional branching** | Arbitrary boolean `while` expression + `validations.skip` |

Together with `loop.results()` feeding back into the `while` condition, the system can simulate any computable function — including mu-recursion (search until an unpredictable condition is met).

## Examples

### Accumulator (sum 1+2+3+4 = 10)

<div v-pre>

```yaml
run:
  loop:
    while: "loop.index() < 4"
    maxIterations: 100
  expr:
    - "{{ set('sum', int(default(get('sum'), 0)) + loop.count()) }}"
  apiResponse:
    success: true
    response:
      partial_sum: "{{ get('sum') }}"
```

</div>

### State-Machine Phase Transition

<div v-pre>

```yaml
run:
  loop:
    while: "int(default(get('phase'), 0)) < 3"
    maxIterations: 10
  expr:
    - "{{ set('phase', int(default(get('phase'), 0)) + 1) }}"
  apiResponse:
    success: true
    response:
      phase: "{{ get('phase') }}"
```

</div>

### Conditional Early Exit (flag-based)

<div v-pre>

```yaml
run:
  loop:
    while: "get('done') == nil"
    maxIterations: 100
  exec:
    command: "check-condition.sh"
  expr:
    - "{{ get('execResource').exitCode == 0 ? set('done', true) : set('noop', 0) }}"
  apiResponse:
    success: true
    response:
      iterations: "{{ loop.count() }}"
```

</div>

### Collect N Results

<div v-pre>

```yaml
run:
  loop:
    while: "len(loop.results()) < 5"
    maxIterations: 50
  chat:
    model: llama3.2:1b
    prompt: "Generate item {{ loop.count() }}"
  apiResponse:
    success: true
    response:
      item: "{{ get('chatResource') }}"
```

</div>

### Downstream Resource Reads Loop Output

A resource that runs a loop and a downstream resource that reads the final state:

<div v-pre>

```yaml
# resources/compute.yaml
metadata:
  actionId: compute
run:
  loop:
    while: "loop.index() < 3"
    maxIterations: 10
  expr:
    - "{{ set('computed', loop.count()) }}"

---
# resources/respond.yaml
metadata:
  actionId: respond
  requires: [compute]
run:
  apiResponse:
    success: true
    response:
      value: "{{ get('computed') }}"   # reads final value set by the loop
```

</div>

## When to Use `loop` vs `items`

| Use `loop` when… | Use `items` when… |
|------------------|------------------|
| Number of iterations is not known in advance | You have a fixed list to process |
| Termination depends on runtime state | You want to iterate over a pre-computed array |
| You need mutable accumulation across iterations | Each item is independent |
| Implementing search / retry / polling patterns | Batch processing of a dataset |

## Next Steps

- [Items Iteration](items) — Fixed-list iteration
- [Expressions](expressions) — Expression syntax
- [Expression Functions Reference](expression-functions-reference) — Complete function list
- [Validation and Control Flow](validation-and-control) — Skip conditions, preflight checks
