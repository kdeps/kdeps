# Optional {{ }} Braces

This document explains how `{{ }}` braces work in kdeps expressions.

## TL;DR

- **Direct values**: `{{ }}` is **optional** ✅
- **String interpolation**: `{{ }}` is **required** ✅
- **Backward compatible**: Old syntax still works ✅

## Syntax Rules

| Context | Braces | Example |
|---------|--------|---------|
| **Direct value** | Optional | `prompt: get('q')` or `prompt: {{ get('q') }}` |
| **String interpolation** | Required | `message: "Hello {{name}}"` |
| **Literal value** | Not needed | `age: 25` |

## Examples

### ✅ Direct Expressions (Braces Optional)

You can write expressions **without** `{{ }}`:

```yaml
# Function calls
prompt: get('q')
count: get('items').length
data: file('config.json')

# Dot notation
email: user.email
name: user.profile.name
age: user.profile.age

# Arithmetic
total: price * quantity
average: sum / count
doubled: score * 2

# Comparisons
isAdult: age >= 18
isValid: status == "active"
hasAccess: premium || trial

# Logical operations
eligible: age >= 18 && verified
active: !disabled
```

### ✅ String Interpolation (Braces Required)

When interpolating within strings, you **must** use `{{ }}`:

```yaml
# Single interpolation
message: "Hello {{name}}"
title: "Welcome {{username}}"

# Multiple interpolations
greeting: "Hello {{first}} {{last}}, welcome!"
status: "User {{username}} is {{status}}"

# Mixed text and expressions
summary: "User {{user.name}} scored {{score}} points"
calculation: "Result: {{x}} + {{y}} = {{x + y}}"

# With function calls
result: "Query result: {{get('q')}}"
timestamp: "Generated at {{info('current_time')}}"
```

### ✅ Backward Compatibility

The old syntax with `{{ }}` still works everywhere:

```yaml
# Both syntaxes work identically

# Without braces (new, cleaner)
prompt: get('q')
email: user.email
total: price * quantity

# With braces (old, still valid)
prompt: {{ get('q') }}
email: {{user.email}}
total: {{ price * quantity }}
```

## How It Works

### Detection

The parser automatically detects the expression type:

1. **Contains `{{ }}`?**
   - → `ExprTypeInterpolated` (may be string interpolation)
   
2. **Looks like an expression?**
   - Has function calls (`get()`, `info()`, etc.)
   - Has operators (`+`, `-`, `>`, `==`, etc.)
   - Has property access (`user.email`)
   - → `ExprTypeDirect`
   
3. **Otherwise:**
   - → `ExprTypeLiteral` (plain value)

### Evaluation

- **Direct expressions** are evaluated using expr-lang
- **Interpolated strings** process each `{{ }}` block:
  - Try mustache variable lookup first
  - Fall back to expr-lang evaluation
- **Literals** are returned as-is

## When to Use What

### Use WITHOUT Braces (Cleaner)

```yaml
# When assigning a direct value
email: user.email
count: items.length
valid: age >= 18
result: get('data')
total: price * quantity
```

**Benefits:**
- Less typing
- Cleaner appearance
- Obvious it's a direct value

### Use WITH Braces (For Interpolation)

```yaml
# When building strings with dynamic values
message: "Hello {{name}}, you have {{count}} items"
url: "https://api.example.com/users/{{userId}}"
status: "{{username}} is {{status}}"
summary: "Total: {{price * quantity}}"
```

**When required:**
- Interpolating within strings
- Multiple values in one string
- Mixing static text with dynamic values

## Common Patterns

### API Responses

```yaml
run:
  apiResponse:
    # Without braces for direct values
    user_email: user.email
    user_name: user.name
    user_age: user.age
    is_adult: user.age >= 18
    
    # With braces for messages
    message: "Welcome {{user.name}}!"
    summary: "User {{user.name}} ({{user.email}})"
```

### Conditional Logic

```yaml
run:
  preflightCheck:
    # Without braces for conditions
    validations:
      - get('q') != ''
      - user.age >= 18
      - status == 'active'
    
    # With braces for error messages
    error:
      message: "User {{user.name}} does not meet requirements"
```

### Data Transformation

```yaml
run:
  transform:
    # Without braces for calculations
    total: price * quantity
    discount: total * 0.1
    final: total - discount
    
    # With braces for formatted output
    display: "Total: ${{total}}, Discount: ${{discount}}"
```

## Technical Details

### Parser Detection

The parser uses several heuristics to detect expressions:

1. **Function patterns**: `get()`, `set()`, `info()`, etc.
2. **Operators**: `!=`, `==`, `>=`, `&&`, `||`, arithmetic
3. **Property access**: `user.email`, `data.items[0]`
4. **Literals excluded**: URLs, MIME types, auth tokens

### Evaluator Behavior

```go
switch expression.Type {
case domain.ExprTypeLiteral:
    return expression.Raw, nil  // Return as-is
    
case domain.ExprTypeDirect:
    return e.evaluateDirect(expression.Raw, env)  // Eval with expr-lang
    
case domain.ExprTypeInterpolated:
    return e.evaluateInterpolated(expression.Raw, env)  // Process {{ }} blocks
}
```

## Migration Guide

### From Old Syntax

No migration needed! Your existing code continues to work:

```yaml
# Old code with braces - still works
prompt: "{{ get('q') }}"
email: "{{user.email}}"
```

### To New Syntax (Optional)

You can gradually adopt the cleaner syntax:

```yaml
# Old
prompt: "{{ get('q') }}"

# New (optional, cleaner)
prompt: get('q')

# But keep braces for interpolation
message: "Hello {{name}}"
```

## Best Practices

### ✅ Do

```yaml
# Use without braces for direct values
email: user.email
count: items.length
valid: age >= 18

# Use with braces for string interpolation
message: "Hello {{name}}"
summary: "Total: {{count}} items"
```

### ❌ Don't

```yaml
# Don't add unnecessary braces to direct values
email: "{{user.email}}"        # Unnecessary quotes and braces
count: "{{ items.length }}"    # Unnecessary

# Simpler:
email: user.email
count: items.length
```

### ⚠️ Watch Out

```yaml
# Without braces, this is literal text
message: Hello name           # Literal: "Hello name"

# With braces, this interpolates
message: "Hello {{name}}"     # Interpolates: "Hello John"

# Function calls are always detected
result: get('data')           # Expression (works)
result: "Result: get('data')" # Also expression (detected!)
```

## FAQ

### Q: Do I need to update my existing workflows?

**A:** No! All existing syntax continues to work. The `{{ }}` braces are still valid everywhere.

### Q: When should I use `{{ }}`?

**A:** Required for string interpolation. Optional for direct values.

### Q: Can I mix both styles?

**A:** Yes! Use what makes sense for each case:
```yaml
email: user.email                    # Direct (no braces)
message: "Welcome {{user.name}}!"    # Interpolation (braces)
```

### Q: What about backward compatibility?

**A:** 100% backward compatible. All existing code works unchanged.

### Q: How do I know if I need braces?

**A:** Ask: "Am I building a string with dynamic values?" 
- Yes → Use braces for interpolation
- No → Braces are optional

## See Also

- [examples/optional-braces/](../examples/optional-braces/) - Working examples
- [examples/mustache-expressions/](../examples/mustache-expressions/) - Mustache-style variables
- [examples/hybrid-expressions/](../examples/hybrid-expressions/) - Mixing syntaxes
- [CONTROL_FLOW.md](./CONTROL_FLOW.md) - Control flow patterns
