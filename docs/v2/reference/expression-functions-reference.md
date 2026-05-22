# Expression Functions Reference

Every function available in kdeps expressions -- usable in any field that supports <span v-pre>`{{ }}`</span> interpolation or in `expr` blocks.

## Core Functions

The four functions used in almost every resource: read data, write data, access uploaded files, and read request metadata.

### get(key, typeHint?)
Retrieves data from any source.

- **key**: The key to look up.
- **typeHint** (optional): Force a specific source (`param`, `header`, `session`, `memory`, `item`, `env`, `file`, `filepath`, `filetype`).

**Examples:**
```yaml
get('q')                     # Auto-detect source
get('Authorization', 'header') # Get from headers
get('user_id', 'session')    # Get from session
get('API_KEY', 'env')        # Get environment variable
```

### set(key, value, storage?)
Stores data in memory or session.

- **key**: The key to store.
- **value**: The value to store.
- **storage** (optional): Storage type (`memory` or `session`). Default is `memory`.

**Examples:**
```yaml
set('count', 1)              # Store in memory (request-scoped)
set('user', data, 'session') # Store in session (persistent)
```

### file(pattern, selector?)
Accesses uploaded files or local files.

- **pattern**: Glob pattern or file name.
- **selector** (optional): `first`, `last`, `count`, `all`, `mime:<type>`.

**Examples:**
```yaml
file('*.jpg')                # Get all JPG files content
file('*.pdf', 'first')       # Get content of first PDF
file('*', 'count')           # Get total file count
```

### info(field)
Retrieves request metadata.

- **field**: One of `ID`, `IP`, `timestamp`, `path`, `method`, `sessionId`, `filecount`, `files`, `filetypes`.

**Examples:**
```yaml
info('ID')                   # Get request ID
info('IP')                   # Get client IP
```

## Data Handling Functions

Utilities for converting, inspecting, and safely traversing data structures.

### json(data)
Converts data to a JSON string.

- **data**: The data to stringify.

**Examples:**
```yaml
json(get('userData'))        # Convert object to JSON string
```

### safe(obj, path)
Safely accesses nested properties without panicking on nil values.

- **obj**: The object to access.
- **path**: Dot-notation path to the property.

**Examples:**
```yaml
safe(user, "profile.address.city") # Returns city or nil if path invalid
```

### debug(obj)
Returns a pretty-printed JSON string representation of an object for debugging.

- **obj**: The object to inspect.

**Examples:**
```yaml
debug(get('httpResponse'))   # Inspect complex object structure
```

### default(value, fallback)
Returns a fallback value if the primary value is nil or empty.

- **value**: The value to check.
- **fallback**: The value to return if primary is nil/empty.

**Examples:**
```yaml
default(get('limit'), 10)    # Return 10 if limit is missing
```

## Input/Output Functions

Explicit accessors for request inputs and resource outputs -- use these when `get()` auto-detection is ambiguous.

### input(name, type?)
Accesses input data (similar to `get` but strictly for inputs).

- **name**: Input name.
- **type** (optional): `param`, `header`, `body`.

**Examples:**
```yaml
input('q')                   # Get query param
input('body')                # Get request body
```

### output(resourceId)
Accesses the output of a completed resource.

- **resourceId**: The actionId of the resource.

**Examples:**
```yaml
output('llmResource')        # Get LLM output
```

## Iteration Functions

Available inside `items:` blocks to access the state of the current loop iteration.

### item object
Accesses current iteration context. `item` is an object -- call its methods to read iteration state.

**Methods:**
```yaml
item.current()   # Current item value
item.prev()      # Previous item (nil on first iteration)
item.next()      # Next item (nil on last iteration)
item.index()     # Current index (0-based)
item.count()     # Total items count
item.values()    # All items as an array
```

## Session Functions

Access session-scoped data set with `set('key', val, 'session')` in any resource.

### session()
Returns the entire session data object.

**Examples:**
```yaml
session()                    # Get all session data
```

## Array Operations

### filter(array, predicate)
Filters an array by a predicate expression. Use `.` to reference the current element.

```yaml
# resources/example.yaml
after:
  - set('activeUsers', filter(get('users'), .status == 'active'))
  - set('premiumActive', filter(get('users'), .status == 'active' && .tier == 'premium'))
  - set('expensiveItems', filter(get('products'), .price > 100))
```

### map(array, expression)
Transforms each element in an array.

```yaml
# resources/example.yaml
after:
  - set('userNames', map(get('users'), .name))
  - set('emails', map(get('users'), .email))
  - set('pricesWithTax', map(get('items'), .price * 1.1))
```

### Aggregation

```yaml
# resources/example.yaml
after:
  - set('total', sum(get('prices')))
  - set('minPrice', min(get('prices')))
  - set('maxPrice', max(get('prices')))
  - set('avgPrice', sum(get('prices')) / len(get('prices')))
```

### slice(array, start, end)
Extracts a sub-array. Negative indices count from the end.

```yaml
# resources/example.yaml
after:
  - set('firstFive', slice(get('items'), 0, 5))
  - set('lastTen', slice(get('items'), -10, len(get('items'))))
```

### first(array) / last(array)
Returns the first or last element of an array.

```yaml
# resources/example.yaml
after:
  - set('firstItem', first(get('items')))
  - set('lastItem', last(get('items')))
```

### len(value)
Returns the length of an array or string.

```yaml
# resources/example.yaml
after:
  - set('itemCount', len(get('items')))
  - set('textLength', len(get('text')))
```

## String Operations

### Case Conversion

```yaml
# resources/example.yaml
after:
  - set('lowercase', lower(get('text')))
  - set('uppercase', upper(get('text')))
  - set('trimmed', trim(get('text')))
```

### Splitting & Joining

```yaml
# resources/example.yaml
after:
  - set('words', split(get('csv'), ','))
  - set('lines', split(get('text'), '\n'))
  - set('commaSeparated', join(get('items'), ', '))
```

### Replacing

```yaml
# resources/example.yaml
after:
  - set('replaced', replace(get('text'), 'old', 'new'))
```

### String Matching

`contains`, `startsWith`, `endsWith`, and `matches` are infix operators, not functions.

```yaml
# resources/example.yaml
after:
  - set('hasKeyword', get('text') contains 'important')
  - set('isUrl', get('url') startsWith 'https://')
  - set('isImage', get('filename') endsWith '.jpg')
  - set('isEmail', get('email') matches '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$')
```

## Type Conversion

### type(value)
Returns the type as a string: `"string"`, `"int"`, `"float"`, `"bool"`, `"array"`, `"map"`, `"nil"`.

```yaml
# resources/example.yaml
after:
  - set('valueType', type(get('value')))
```

### Casting Functions

```yaml
# resources/example.yaml
after:
  - set('age', int(get('ageString')))       # "123" -> 123
  - set('price', float(get('priceString'))) # "3.14" -> 3.14
  - set('idString', string(get('id')))      # 42 -> "42"
```

## Date & Time

### info('timestamp')
Returns the current time as an RFC3339 string (e.g. `2024-12-25T14:30:00Z`). Use this for timestamps in responses, logging, and audit fields.

```yaml
# resources/example.yaml
after:
  - set('ts', info('timestamp'))
```

### now()
Returns the current time as a `time.Time` value. Useful with comparison operators or passing to `date()` for parsing.

```yaml
# resources/example.yaml
after:
  - set('currentTime', now())
```

## Conditional Logic

### Ternary Operator

```yaml
# resources/example.yaml
after:
  - set('status', get('score') >= 70 ? 'pass' : 'fail')
  - set('discount', get('isPremium') ? 0.2 : 0.1)
```

### Null Coalescing

The `??` operator returns the right-hand value when the left-hand is nil or empty string.

```yaml
# resources/example.yaml
after:
  - set('name', get('name') ?? 'Anonymous')
  - set('limit', get('limit') ?? 10)
```

## Operator Precedence

Expressions evaluate left-to-right with this precedence (highest to lowest):

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

## Best Practices

- **Use parentheses for clarity** -- `(a + b) * c` is clearer than relying on precedence
- **Break complex expressions** into multiple statements for readability
- **Validate before processing** -- check types and null values before operations
- **Keep expressions simple** -- complex logic belongs in Python resources

## See Also

- [Expressions Guide](/concepts/expressions) -- where expressions are used and basic syntax
- [Validation & Control Flow](/concepts/validation-and-control) -- skip, check, and error handling
- [Inline Resource Blocks](/reference/expr-blocks) -- `before:` and `after:` expression blocks