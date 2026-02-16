# Optional Braces Example

This example demonstrates that `{{ }}` braces are **optional** for direct expressions but **required** for string interpolation.

## Key Features

### ✅ Direct Expressions (Braces Optional)

You can write expressions **without** `{{ }}`:

```yaml
# Function calls
prompt: get('q')
count: get('items').length

# Dot notation
email: user.email
name: user.profile.name

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
message: "Hello {{name}}"

# Multiple interpolations
greeting: "Hello {{first}} {{last}}, welcome!"

# Mixed text and expressions
summary: "User {{user.name}} scored {{score}} points"

# With function calls
result: "Query result: {{get('q')}}"
```

### ✅ Backward Compatibility

The old syntax with `{{ }}` still works everywhere:

```yaml
# Both syntaxes work
prompt: get('q')           # New: without braces
prompt: {{ get('q') }}     # Old: with braces

email: user.email          # New: without braces
email: {{user.email}}      # Old: with braces
```

## Syntax Rules

| Context | Braces | Example |
|---------|--------|---------|
| **Direct value** | Optional | `prompt: get('q')` |
| **String interpolation** | Required | `message: "Hello {{name}}"` |
| **Literal value** | Not needed | `age: 25` |

## When to Use What

### Use WITHOUT Braces (Cleaner)
```yaml
# When assigning a direct value
email: user.email
count: items.length
valid: age >= 18
result: get('data')
```

### Use WITH Braces (For Interpolation)
```yaml
# When building strings with dynamic values
message: "Hello {{name}}, you have {{count}} items"
url: "https://api.example.com/users/{{userId}}"
status: "{{username}} is {{status}}"
```

## Benefits

1. **Cleaner Syntax**: Less typing for direct expressions
2. **More Readable**: Obvious when interpolation is happening
3. **Backward Compatible**: Old syntax still works
4. **Flexible**: Choose the style that fits your needs

## See Also

- [examples/mustache-expressions/](../mustache-expressions/) - Mustache-style variables
- [examples/hybrid-expressions/](../hybrid-expressions/) - Mixing function calls and dot notation
- [examples/control-flow/](../control-flow/) - Control flow patterns
