# Mustache Expressions Example

This example demonstrates the unified expression syntax in kdeps.

## Overview

kdeps expressions now support **unified syntax** - use either style anywhere:

### Simple Variables (Mustache-style)
```yaml
prompt: "{{q}}"           # No spaces
prompt: "{{ q }}"         # With spaces - both work!
name: "{{user.name}}"     # Dot notation
```

### Functions & Complex Expressions (expr-lang)
```yaml
timestamp: "{{ info('current_time') }}"
result: "{{ get('count') + 10 }}"
status: "{{ score > 80 ? 'Pass' : 'Fail' }}"
```

### Mix Them Together!
```yaml
message: "Hello {{name}}, time is {{ info('time') }}"
```

## Why This is Better

- **No whitespace distinction**: `{{var}}` and `{{ var }}` both work
- **Natural mixing**: Use simple vars and functions together
- **Smart detection**: System tries mustache first, falls back to expr-lang
- **Fully backward compatible**: All existing syntax works

## How It Works

For each `{{ }}` block:
1. Check if it's a simple variable (no operators, no functions)
2. If yes, try mustache lookup (simple and fast)
3. If no (or not found in mustache), use expr-lang (powerful)

## Running This Example

```bash
cd examples/mustache-expressions
kdeps run workflow.yaml --dev
```

Then test with:

```bash
curl "http://localhost:16395/api/demo?q=test&name=Alice"
```

## Examples

### Simple Variables Work Everywhere
```yaml
# All of these work now:
name: "{{name}}"
name: "{{ name }}"
name: "{{user.name}}"
name: "{{ user.name }}"
```

### Mix Variable Access and Function Calls
```yaml
# Naturally combine both:
greeting: "Hello {{name}}, you have {{count}} items"
timestamp: "Generated at {{ info('current_time') }}"
result: "Score: {{ get('score') * 2 }}"

# All in one line:
message: "{{username}} scored {{ get('points') + 10 }} at {{ info('time') }}"
```

### No More Thinking About Syntax
```yaml
# Just write what makes sense:
simple: "{{var}}"              # Simple variable
complex: "{{ get('x') + 10 }}" # Complex expression
mixed: "{{name}} at {{ info('time') }}" # Mixed!
```

## Benefits

1. **Simpler** - No need to remember spacing rules
2. **Natural** - Mix simple and complex freely
3. **Powerful** - Full expr-lang when needed
4. **Compatible** - All old syntax still works

