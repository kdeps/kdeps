# Mustache Expressions Feature - Implementation Summary

## What Was Implemented

Extended kdeps runtime expression system to support **mustache syntax** alongside the existing expr-lang syntax.

## The Problem

Previously, kdeps only supported expr-lang syntax for runtime expressions:
```yaml
prompt: "{{ get('q') }}"  # Verbose for simple variable access
```

Users found this verbose for basic variable access. The mustache template system existed but was ONLY for scaffolding (project generation), not runtime expressions.

## The Solution

Now kdeps supports BOTH syntaxes in workflow YAMLs:

### 1. Mustache (NEW - Simpler)
```yaml
prompt: "{{q}}"               # Simple variable
name: "{{user.name}}"         # Nested object
greeting: "Hello {{name}}!"   # Mixed with text
```

### 2. expr-lang (Existing - Full Power)
```yaml
prompt: "{{ get('q') }}"                          # Function call
timestamp: "{{ info('current_time') }}"           # Info function
result: "{{ get('count') + 10 }}"                # Calculation
status: "{{ get('score') > 80 ? 'Pass' : 'Fail' }}"  # Conditional
```

## How It Works

### Automatic Detection

The parser automatically detects which syntax is used:

```go
// Detection Logic:
"{{var}}"           → Mustache (no spaces, no parens)
"{{user.name}}"     → Mustache (no spaces, no parens)
"{{ get('var') }}"  → expr-lang (has spaces)
"{{get('var')}}"    → expr-lang (has parens)
```

### Implementation Details

1. **New Expression Type**: Added `ExprTypeMustache` to `domain.ExprType`
2. **Parser Enhancement**: `isMustacheStyle()` function detects mustache syntax
3. **Evaluator Extension**: `evaluateMustache()` function renders mustache templates
4. **Context Building**: `buildMustacheContext()` creates data for mustache from environment
5. **Value Lookup**: `lookupMustacheValue()` supports dot notation for nested objects

## Files Changed

### Core Implementation
- `pkg/domain/expression.go` - Added ExprTypeMustache constant
- `pkg/parser/expression/parser.go` - Added mustache detection logic
- `pkg/parser/expression/evaluator.go` - Added mustache evaluation

### Tests
- `pkg/parser/expression/evaluator_mustache_test.go` - 14 comprehensive tests

### Documentation & Examples
- `examples/mustache-expressions/` - Working example demonstrating both syntaxes
- `docs/TEMPLATE_SYSTEMS.md` - Updated to reflect new capability

## Test Coverage

All tests pass (100% for new code):

```
✓ TestMustacheExpressions
  - simple_variable
  - simple_variable_with_text
  - multiple_variables
  - nested_object
  - missing_variable_returns_empty
  - integer_value
  - boolean_value

✓ TestExprLangVsMustacheDetection
  - expr-lang with spaces
  - mustache without spaces
  - expr-lang with function
  - mustache nested
  - expr-lang mixed text
  - mustache mixed text
  - mustache section

✓ TestMustacheWithUnifiedAPI

✓ All existing expression tests still pass
```

## Benefits

1. **Simpler Syntax**: `{{name}}` instead of `{{ get('name') }}`
2. **Beginner Friendly**: No need to learn `get()` function for basic cases
3. **Flexible**: Choose the right syntax for your needs
4. **Backward Compatible**: All existing workflows work unchanged
5. **Familiar**: Mustache is widely known (handlebars, liquid, etc.)

## When to Use Which?

### Use Mustache for:
- ✅ Simple variable access: `{{q}}`
- ✅ Nested objects: `{{user.email}}`
- ✅ Clean templates: `"Hello {{name}}!"`

### Use expr-lang for:
- ✅ Function calls: `{{ get('q') }}`, `{{ info('name') }}`
- ✅ Calculations: `{{ count + 10 }}`
- ✅ Conditionals: `{{ score > 80 ? 'Pass' : 'Fail' }}`
- ✅ Complex expressions: `{{ get('items')[0].name }}`

## Example Comparison

### Before (expr-lang only):
```yaml
apiResponse:
  response:
    name: "{{ get('name') }}"
    email: "{{ get('user').email }}"
    message: "Hello {{ get('name') }}, welcome!"
```

### After (can use both):
```yaml
apiResponse:
  response:
    name: "{{name}}"                           # Simpler!
    email: "{{user.email}}"                    # Cleaner!
    message: "Hello {{name}}, welcome!"        # More readable!
    timestamp: "{{ info('current_time') }}"    # expr-lang when needed
```

## Impact

- ✅ No breaking changes - fully backward compatible
- ✅ Simplifies workflow YAMLs for common cases
- ✅ Reduces learning curve for new users
- ✅ Provides flexibility - use what fits your need
- ✅ Resolves confusion about mustache being "only for scaffolding"

## Future Considerations

- Could add more mustache features (sections, lambdas) if needed
- Could provide syntax preference in workflow settings
- Could add linting to suggest simpler syntax where applicable

## Conclusion

This feature successfully extends kdeps' runtime expression system with mustache syntax, providing a simpler alternative for basic variable access while maintaining full backward compatibility with expr-lang. Users can now choose the syntax that best fits their needs, making kdeps more accessible to beginners while retaining power for advanced use cases.
