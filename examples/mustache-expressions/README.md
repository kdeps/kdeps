# Mustache Expressions Example

This example demonstrates the new mustache-style expression syntax in kdeps.

## Overview

kdeps now supports **two syntaxes** for expressions in workflow YAMLs:

### 1. expr-lang (Full Power)
```yaml
prompt: "{{ get('q') }}"
timestamp: "{{ info('current_time') }}"
result: "{{ get('count') + 10 }}"
```

### 2. Mustache (Simpler)
```yaml
prompt: "{{q}}"
timestamp: "{{current_time}}"
name: "{{user.name}}"
```

## Why Mustache?

- **Simpler syntax** for basic variable access
- **No function calls needed** for simple cases
- **Familiar** to users of mustache, handlebars, etc.
- **Backward compatible** - expr-lang still works!

## Detection

kdeps automatically detects which syntax you're using:
- `{{ get('q') }}` → expr-lang (has spaces, function call)
- `{{q}}` → mustache (no spaces, simple variable)

## Running This Example

```bash
cd examples/mustache-expressions
kdeps run workflow.yaml --dev
```

Then test with:

```bash
# Test mustache expressions
curl "http://localhost:16395/api/demo?q=test&name=Alice"

# You'll see both syntaxes work:
# - query_mustache: uses {{q}}
# - query_exprLang: uses {{ get('q') }}
# Both return the same value!
```

## Examples

### Simple Variables
```yaml
# Mustache (simpler)
message: "{{name}}"

# expr-lang (traditional)
message: "{{ get('name') }}"
```

### Nested Objects
```yaml
# Mustache supports dot notation
email: "{{user.email}}"
city: "{{user.address.city}}"

# expr-lang requires explicit path
email: "{{ get('user').email }}"
```

### Mixed Text
```yaml
# Both work with text interpolation
greeting_mustache: "Hello {{name}}, welcome!"
greeting_exprLang: "Hello {{ get('name') }}, welcome!"
```

### When to Use Which?

Use **mustache** for:
- Simple variable access
- Nested object fields
- Clean, readable templates

Use **expr-lang** for:
- Function calls: `get()`, `info()`, `env()`
- Calculations: `{{ get('count') + 10 }}`
- Conditionals: `{{ get('score') > 80 ? 'Pass' : 'Fail' }}`
- Complex expressions

## Benefits

1. **Simpler for beginners** - no need to learn `get()` function
2. **Less verbose** - `{{name}}` vs `{{ get('name') }}`
3. **Flexible** - use what fits your need
4. **Backward compatible** - existing workflows unchanged
