# Advanced Expression Features

KDeps expressions support advanced features beyond basic operators and functions, including array operations, type conversions, and utility functions.

## Array Operations

### Filtering Arrays

Filter arrays based on conditions:

```yaml
run:
  expr:
    # Filter active items
    - set('activeUsers', filter(get('users'), .status == 'active'))
    
    # Filter by multiple conditions
    - set('premiumActive', filter(get('users'), .status == 'active' && .tier == 'premium'))
    
    # Filter items with price > 100
    - set('expensiveItems', filter(get('products'), .price > 100))
```

### Mapping Arrays

Transform arrays by extracting or computing values:

```yaml
run:
  expr:
    # Extract names from user objects
    - set('userNames', map(get('users'), .name))
    
    # Extract emails
    - set('emails', map(get('users'), .email))
    
    # Compute derived values
    - set('pricesWithTax', map(get('items'), .price * 1.1))
```

### Array Aggregation

```yaml
run:
  expr:
    # Sum of numbers
    - set('total', sum(get('prices')))
    
    # Minimum value
    - set('minPrice', min(get('prices')))
    
    # Maximum value
    - set('maxPrice', max(get('prices')))
    
    # Average (sum / length)
    - set('avgPrice', sum(get('prices')) / len(get('prices')))
```

### Array Slicing

```yaml
run:
  expr:
    # First 5 items
    - set('firstFive', slice(get('items'), 0, 5))
    
    # Last 10 items (using negative indexing)
    - set('lastTen', slice(get('items'), -10, len(get('items'))))
    
    # Middle section
    - set('middle', slice(get('items'), 5, 15))
```

## String Operations

### Case Conversion

```yaml
run:
  expr:
    - set('lowercase', lower(get('text')))
    - set('uppercase', upper(get('text')))
    - set('trimmed', trim(get('text')))
```

### String Manipulation

```yaml
run:
  expr:
    # Split string into array
    - set('words', split(get('csv'), ','))
    - set('lines', split(get('text'), '\n'))
    
    # Join array into string
    - set('commaSeparated', join(get('items'), ', '))
    - set('newlineSeparated', join(get('lines'), '\n'))
    
    # Replace text
    - set('replaced', replace(get('text'), 'old', 'new'))
```

### String Matching

```yaml
run:
  expr:
    # Check if contains substring
    - set('hasKeyword', contains(get('text'), 'important'))
    
    # Check if starts with
    - set('isUrl', startsWith(get('url'), 'https://'))
    
    # Check if ends with
    - set('isImage', endsWith(get('filename'), '.jpg'))
    
    # Regex match
    - set('isEmail', matches(get('email'), '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$'))
```

## Type Conversion

### Type Checking

```yaml
run:
  expr:
    - set('valueType', type(get('value')))
    # Returns: "string", "number", "boolean", "array", "object", "null"
```

### Type Conversion

```yaml
run:
  expr:
    # Convert to integer
    - set('age', int(get('ageString')))
    
    # Convert to float
    - set('price', float(get('priceString')))
    
    # Convert to string
    - set('idString', string(get('id')))
    
    # Convert to boolean
    - set('isEnabled', bool(get('enabled')))
```

**Note:** Type conversions handle common cases:
- `int("123")` → `123`
- `float("3.14")` → `3.14`
- `string(42)` → `"42"`
- `bool("true")` → `true`
- `bool(1)` → `true`
- `bool(0)` → `false`

## Date and Time

### Current Time

```yaml
run:
  expr:
    - set('now', now())
    # Returns: ISO 8601 timestamp string
```

### Date Formatting

```yaml
run:
  expr:
    # Format current time
    - set('date', format(now(), '2006-01-02'))
    - set('datetime', format(now(), '2006-01-02 15:04:05'))
    - set('timestamp', format(now(), '2006-01-02T15:04:05Z'))
```

**Date Format Reference:**
- `2006-01-02` → `2024-12-25`
- `2006-01-02 15:04:05` → `2024-12-25 14:30:00`
- `2006-01-02T15:04:05Z` → `2024-12-25T14:30:00Z`
- `01/02/2006` → `12/25/2024`

## Array Utilities

### First and Last

```yaml
run:
  expr:
    - set('firstItem', first(get('items')))
    - set('lastItem', last(get('items')))
```

### Length

```yaml
run:
  expr:
    # Array length
    - set('itemCount', len(get('items')))
    
    # String length
    - set('textLength', len(get('text')))
```

## Conditional Logic

### Ternary Operator

```yaml
run:
  expr:
    - set('status', get('score') >= 70 ? 'pass' : 'fail')
    - set('discount', get('isPremium') ? 0.2 : 0.1)
```

### Null Coalescing

```yaml
run:
  expr:
    # Default value if null
    - set('name', get('name') ?? 'Anonymous')
    - set('limit', get('limit') ?? 10)
```

**Note:** The `??` operator returns the right-hand value if the left-hand value is `nil` or empty string.

## Complex Examples

### Data Transformation Pipeline

```yaml
run:
  expr:
    # Filter and transform
    - set('activeUsers', filter(get('users'), .status == 'active'))
    - set('userEmails', map(get('activeUsers'), .email))
    - set('emailList', join(get('userEmails'), ', '))
    
    # Process and aggregate
    - set('totalValue', sum(map(get('items'), .price)))
    - set('avgValue', get('totalValue') / len(get('items')))
```

### Validation with Expressions

```yaml
run:
  expr:
    # Complex validation logic
    - set('isValidEmail', matches(get('email'), '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$'))
    - set('isValidAge', get('age') >= 18 && get('age') <= 120)
    - set('hasRequiredFields', get('name') != nil && get('email') != nil)
    - set('isValid', get('isValidEmail') && get('isValidAge') && get('hasRequiredFields'))
  
  preflightCheck:
    validations:
      - get('isValid')
    error:
      code: 400
      message: Invalid input data
```

### Dynamic Configuration

```yaml
run:
  expr:
    # Build configuration from multiple sources
    - set('config', {
        "timeout": default(get('TIMEOUT', 'env'), 30),
        "retries": default(get('RETRIES', 'env'), 3),
        "model": default(get('model'), 'llama3.2:1b'),
        "debug": default(get('DEBUG', 'env'), false)
      })
    
    # Conditional configuration
    - set('cacheTTL', get('env') == 'production' ? '1h' : '5m')
```

## Expression Evaluation Order

Expressions are evaluated left-to-right with operator precedence:

1. **Parentheses** - `(a + b) * c`
2. **Unary operators** - `!`, `-`
3. **Multiplicative** - `*`, `/`, `%`
4. **Additive** - `+`, `-`
5. **Comparison** - `<`, `<=`, `>`, `>=`
6. **Equality** - `==`, `!=`
7. **Logical AND** - `&&`
8. **Logical OR** - `||`
9. **Ternary** - `? :`
10. **Null coalescing** - `??`

## Best Practices

1. **Use parentheses for clarity** - `(a + b) * c` is clearer than relying on precedence
2. **Break complex expressions** - Use multiple `expr` statements for readability
3. **Validate before processing** - Check types and null values before operations
4. **Use helper functions** - `default()`, `safe()`, `json()` for common patterns
5. **Keep expressions simple** - Complex logic belongs in Python resources

## See Also

- [Expressions](expressions.md) - Basic expression syntax
- [Expression Helpers](expression-helpers.md) - Helper functions
- [Unified API](unified-api.md) - Data access functions
