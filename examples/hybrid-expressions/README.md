# Hybrid Expressions Example

This example demonstrates **hybrid expressions** that mix:
- `get()` function calls (from kdeps API)
- Dot notation variable access (`user.email`)
- Arithmetic operators (`*`, `+`, `-`, `/`)
- Comparison operators (`>`, `>=`, `==`)
- Ternary operators (`? :`)

## Key Feature

You can write expressions like:
```
{{get('C') * user.email}}
```

This works because:
1. `get('C')` retrieves a value from kdeps API (memory/session/etc.)
2. `user.email` accesses a nested property from the environment
3. `*` performs multiplication
4. All within a single expression!

## Examples in this Workflow

### Simple Hybrid
```yaml
"doubled_age": {{get('multiplier') * user.age}}
```
- `get('multiplier')` → retrieves multiplier from memory
- `user.age` → accesses age property from user object
- Result: `2 * 30 = 60`

### Complex Calculation
```yaml
"personalized_price": {{(get('basePrice') * get('multiplier')) - user.discount}}
```
- Multiple `get()` calls
- Parentheses for order of operations
- Dot notation for object property
- Result: `(50 * 2) - 15 = 85`

### With Ternary
```yaml
"discount_message": "{{user.premium ? 'Premium discount: ' + user.discount : 'Standard discount'}}"
```
- Conditional logic with `? :`
- String concatenation
- Dot notation access

## How It Works

The expr-lang evaluator:
1. Receives environment variables as a map
2. Supports native dot notation for map property access
3. Provides kdeps functions like `get()`, `info()`, `env()`
4. Allows mixing them all in one expression

## Running the Example

```bash
kdeps run examples/hybrid-expressions/workflow.yaml
```

## Expected Output

```json
{
  "user_name": "Alice",
  "user_age": 30,
  "doubled_age": 60,
  "price_with_discount": 35,
  "personalized_price": 85,
  "is_adult": true,
  "discount_message": "Standard discount",
  "computed_total": 750
}
```

## Benefits

1. **Natural syntax** - Write expressions as you think
2. **No conversion needed** - Mix API calls and data access
3. **Full power** - All operators and functions work together
4. **Type safety** - expr-lang handles type conversions
5. **Readable** - Self-documenting expressions

## More Examples

```yaml
# Arithmetic with API and object
price: {{get('basePrice') * quantity.amount}}

# Comparison
eligible: {{user.age >= get('minAge')}}

# Complex nested
total: {{(get('price') * order.items.length) + shipping.cost}}

# With function calls
result: {{get('x') + get('y') * user.profile.score}}
```

## Technical Details

- **Evaluator**: Uses expr-lang library
- **Environment**: All variables passed to evaluator
- **Dot notation**: Native expr-lang feature for maps
- **Functions**: Wrapped kdeps API calls
- **Type handling**: Automatic by expr-lang
