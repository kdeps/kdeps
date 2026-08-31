# Expression operators

All comparison and logical operators available in `validations.check`, `validations.skip`, and any boolean expression context.

*Applies to both workflow mode and agent mode.*

## Comparison operators

| Operator | Description | Example |
|---|---|---|
| `==` | Equal to | `get('status') == 'active'` |
| `!=` | Not equal to | `get('role') != 'admin'` |
| `>` / `gt` | Greater than | `get('age') > 18` |
| `>=` / `gte` | Greater than or equal | `get('score') >= 70` |
| `<` / `lt` | Less than | `get('price') < 100` |
| `<=` / `lte` | Less than or equal | `get('count') <= 10` |

## String operators

| Operator | Description | Example |
|---|---|---|
| `contains` | String contains substring | `get('text') contains 'urgent'` |
| `startsWith` | String starts with prefix | `get('url') startsWith 'https://'` |
| `endsWith` | String ends with suffix | `get('file') endsWith '.pdf'` |
| `matches` | Regex match | `get('email') matches '^[^@]+@[^@]+$'` |

## Array operators

| Operator | Description | Example |
|---|---|---|
| `in` | Value is in array | `get('role') in ['admin', 'mod']` |
| `notIn` | Value is not in array | `get('role') notIn ['banned']` |
| `contains` | Array contains value | `get('tags') contains 'featured'` |
| `len` | Array or string length | `len(get('items')) > 0` |

## Logical operators

| Operator | Description | Example |
|---|---|---|
| `&&` / `and` | Logical AND | `get('age') >= 18 && get('verified')` |
| `\|\|` / `or` | Logical OR | `get('role') == 'admin' \|\| get('role') == 'mod'` |
| `!` / `not` | Logical NOT | `!get('isBanned')` |

## Membership & null

| Operator | Description | Example |
|---|---|---|
| `??` | Null coalescing (return right if left is nil/empty) | `get('name') ?? 'Anonymous'` |
| `?:` | Elvis operator (return left if truthy, else right) | `get('name') ?: 'Unknown'` |
| `? :` | Ternary conditional | `get('score') >= 70 ? 'pass' : 'fail'` |
| `nil` | Check for nil/null | `get('optional') != nil` |

## Operator precedence

Highest to lowest:

1. Parentheses: `(a + b) * c`
2. Unary: `!`, `-`
3. Multiplicative: `*`, `/`, `%`
4. Additive: `+`, `-`
5. Comparison: `<`, `<=`, `>`, `>=`
6. Equality: `==`, `!=`
7. Logical AND: `&&`
8. Logical OR: `||`
9. Ternary: `? :`
10. Null coalescing: `??`

## Usage contexts

### In validations.check (all must pass)

```yaml
# resources/example.yaml
validations:
  check:
    - get('email') != nil
    - get('email') contains '@'
    - len(get('password')) >= 8
  error:
    code: 400
    message: Invalid email or password too short
```

### In validations.skip (any triggers skip)

```yaml
# resources/example.yaml
validations:
  skip:
    - get('q') == ''
    - get('q') == nil
```

### In before:/after: blocks

```yaml
# resources/example.yaml
after:
  - set('isAdmin', get('role') in ['admin', 'superadmin'])
  - set('needsReview', get('amount') > 1000 || get('isNewUser'))
  - set('displayName', get('name') ?? 'Guest')
```

### In template interpolation

<div v-pre>

```yaml
# resources/example.yaml
chat:
  prompt: |
    User is {{ get('age') >= 18 ? 'adult' : 'minor' }}.
    Role: {{ get('role') ?? 'user' }}.
```

</div>

## See also

- [Expression functions reference](/reference/expression-functions-reference) - all functions (get, set, file, info, and more)
- [Expressions guide](/concepts/expressions) - where expressions are used and basic syntax
- [Expression blocks](/reference/expr-blocks) - `before:` and `after:` usage
- [Validation & control flow](/concepts/validation-and-control) - skip vs check logic
