# Hybrid Expressions Example

This example demonstrates **hybrid expressions** that mix:
- `get()` function calls (from kdeps runtime API)
- Arithmetic operators (`*`, `+`, `-`, `/`)
- Comparison operators (`>`, `>=`, `==`)
- Ternary operators (`? :`)

## Key Feature

You can write expressions like:

```yaml
doubled: "{{ get('multiplier') * 2 }}"
```

## Examples in this Workflow

### Arithmetic with API values
```yaml
doubled: "{{ get('value') * 2 }}"
total: "{{ get('price') * get('quantity') }}"
```

### Conditional logic
```yaml
status: "{{ get('score') >= 80 ? 'Pass' : 'Fail' }}"
```

### Default values
```yaml
label: "{{ default(get('name'), 'unknown') }}"
```

## Running the Example

```bash
kdeps run examples/hybrid-expressions/workflow.yaml
```

## How It Works

The expr-lang evaluator:
1. Receives the kdeps context (memory, session, request params)
2. Provides kdeps functions like `get()`, `set()`, `info()`, `default()`
3. Allows arithmetic, comparison, and ternary operators within `{{ }}`

## More Examples

```yaml
# Arithmetic with stored values
price: "{{ get('base_price') * get('quantity') }}"

# Comparison
eligible: "{{ get('score') >= get('min_score') }}"

# Ternary
tier: "{{ get('score') >= 90 ? 'gold' : get('score') >= 70 ? 'silver' : 'bronze' }}"

# Default fallback
label: "{{ default(get('name'), 'anonymous') }}"
```
