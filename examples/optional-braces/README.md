# Optional Braces Example

This example demonstrates that `{{ }}` braces are **optional** for direct expressions but **required** for string interpolation.

## Key Features

### ✅ Direct Expressions (Braces Optional)

You can write expressions **without** `{{ }}`:

```yaml
# Function calls
prompt: get('q')
count: get('items').length

# Arithmetic
total: price * quantity
average: sum / count

# Comparisons
isAdult: age >= 18
isValid: status == "active"
```

### ✅ String Interpolation (Braces Required)

When interpolating within strings, you **must** use `{{ }}`:

```yaml
# Single interpolation
message: "Hello {{ get('name') }}"

# Multiple interpolations
greeting: "Score: {{ get('score') }} out of {{ get('total') }}"

# With function calls
result: "Query result: {{ get('q') }}"
```

## Syntax Rules

| Context | Braces | Example |
|---------|--------|---------|
| **Direct value** | Optional | `prompt: get('q')` |
| **String interpolation** | Required | `message: "Hello {{ get('name') }}"` |
| **Literal value** | Not needed | `age: 25` |

## When to Use What

### Use WITHOUT Braces (Cleaner)
```yaml
# When assigning a direct value
count: get('items').length
valid: age >= 18
result: get('data')
```

### Use WITH Braces (For Interpolation)
```yaml
# When building strings with dynamic values
message: "You have {{ get('count') }} items"
url: "https://api.example.com/users/{{ get('userId') }}"
```

## See Also

- [examples/jinja2-expressions/](../jinja2-expressions/) - Jinja2 YAML preprocessing
- [examples/hybrid-expressions/](../hybrid-expressions/) - Mixing function calls and dot notation
- [examples/control-flow/](../control-flow/) - Control flow patterns
