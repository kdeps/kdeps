# Expressions

Expressions are how you pass data between resources, validate inputs, and run conditional logic. They are powered by [expr-lang](https://expr-lang.org/).

## Where expressions are used

**String interpolation** -- embed an expression inside any string field using <span v-pre>`{{ }}`</span>:

<div v-pre>

```yaml
# resources/example.yaml
chat:
  prompt: "Hello {{ get('name') }}, today is {{ info('timestamp') }}"
```

</div>

**before:/after: blocks** -- a list of statements executed sequentially. Each statement is a bare expression, not wrapped in <span v-pre>`{{ }}`</span>:

```yaml
# resources/example.yaml
after:
  - set('normalized', lower(trim(get('q'))))   # stores a value
  - set('is_admin', get('role') == 'admin')    # boolean
  - set('total', get('price') * get('quantity'))
```

**validations.skip / validations.check / onError.when** -- a list of boolean expressions; any one true is enough:

```yaml
# resources/example.yaml
validations:
  skip:
    - get('q') == ''           # bare expression, evaluated as bool
  check:
    - len(get('password')) >= 8
```

## Standard library

Expressions have access to the full [expr-lang standard library](https://expr-lang.org/docs/language-definition):

- **String**: `trim()`, `lower()`, `upper()`, `split()`, `replace()`, `join()`
- **Math**: `min()`, `max()`, `abs()`, `ceil()`, `floor()`
- **List**: `len()`, `filter()`, `map()`, `first()`, `last()`, `contains()`
- **Type casting**: `int()`, `float()`, `string()`, `bool()`

kdeps adds workflow-specific helpers on top. See the [Expression Functions Reference](/reference/expression-functions-reference) for the full list.
## See Also

- [Expression Functions Reference](/reference/expression-functions-reference) - All kdeps-specific functions
- [Expression Operators](/reference/expression-operators) - Comparison and logic operators
- [Expression Blocks](/reference/expr-blocks) - `before:` / `after:` statement blocks
