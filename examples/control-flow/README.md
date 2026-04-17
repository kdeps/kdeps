# Control Flow Examples

This example demonstrates control flow in kdeps:
- **if-else** via ternary operator (`? :`)
- **and/or** via logical operators (`&&`, `||`, `!`)
- **while loops** via the `loop` resource block (Turing-complete)
- **list operations** via functional expressions (`filter`, `map`, `all`, `any`)

## Conditional Iteration: `loop`

The `loop` block enables unbounded while-loop iteration in a resource:

```yaml
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

- `loop.index()` — current iteration index (0-based)
- `loop.count()` — current iteration count (1-based)
- `loop.results()` — results from all prior iterations (for self-referential termination)
- `set('key', val, 'loop')` / `get('key', 'loop')` — loop-scoped storage

Multiple iterations with `apiResponse` produce a **streaming response** (a slice of per-iteration maps).

## Features Demonstrated

### 1. If-Else (Ternary Operator)

```yaml
# Simple condition
status: {{age >= 18 ? "adult" : "child"}}

# Nested conditions  
category: {{age < 13 ? "child" : (age < 20 ? "teen" : "adult")}}

# With calculations
discount: {{premium ? price * 0.8 : price}}
```

### 2. Logical Operators

```yaml
# AND
eligible: {{age >= 18 && verified}}

# OR
hasAccess: {{premium || trial}}

# NOT
enabled: {{!disabled}}

# Complex
canPurchase: {{(age >= 18 && verified) || admin}}
```

### 3. While Loop

```yaml
# Count to N
loop:
  while: "loop.index() < 10"
  maxIterations: 100

# Accumulate until threshold
loop:
  while: "int(default(get('sum'), 0)) <= 20"
  maxIterations: 100

# Collect N results
loop:
  while: "len(loop.results()) < 3"
  maxIterations: 10
```

### 4. List Operations

```yaml
# Filter (like: for item in items if condition)
adults: {{filter(users, .age >= 18)}}

# Map (like: [item.name for item in items])
names: {{map(users, .name)}}

# All (like: all(item.valid for item in items))
allValid: {{all(items, .valid)}}

# Any (like: any(item.active for item in items))
hasActive: {{any(items, .active)}}
```

## Running the Example

```bash
kdeps run examples/control-flow/workflow.yaml
```

## Learn More

- [Loop Iteration Documentation](../../docs/v2/concepts/loop.md) - Complete loop guide
- [Control Flow Documentation](../../docs/CONTROL_FLOW.md) - Expressions and conditionals
- [expr-lang Documentation](https://expr-lang.org/docs/language-definition) - Full language reference
