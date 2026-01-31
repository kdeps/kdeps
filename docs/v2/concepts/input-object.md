# Input Object

The `input` object provides convenient property-based access to request body data. It's available in all expressions and `expr` blocks.

## Overview

The `input` object is automatically populated with the request body data, allowing you to access fields directly as properties instead of using `get()`.

## Basic Usage

### Property Access



Access request body fields as properties:



<div v-pre>



```yaml

run:

  expr:

    - set('query', input.q)

    - set('userId', input.userId)

    - set('items', input.items)

  

  chat:

    prompt: "User {{ get('userId') }} asked: {{ get('query') }}"

```



</div>

### In String Interpolation

Use `input` directly in interpolated strings:

<div v-pre>

```yaml
run:
  chat:
    prompt: "Hello {{ input.name }}, you asked about {{ input.topic }}"
```

</div>

### Nested Properties

Access nested object properties:

```yaml
run:
  expr:
    - set('city', input.user.address.city)
    - set('email', input.user.email)
```

## Comparison with Unified API

The `input` object is a convenience wrapper around request body data. Both approaches work:

| Input Object | Unified API | Description |
|--------------|------------|-------------|
| `input.field` | `get('field')` | Request body field |
| `input.user.name` | `get('user').name` | Nested property |
| `input.items[0]` | `get('items')[0]` | Array element |

**Example:**

```yaml
# Using input object
run:
  expr:
    - set('name', input.name)
    - set('email', input.user.email)

# Equivalent using Unified API
run:
  expr:
    - set('name', get('name'))
    - set('email', get('user').email)
```

## When to Use

### Use `input` when:
- Accessing request body fields frequently
- Working with nested object structures
- Writing concise property access code

### Use `get()` when:
- Need to access query parameters, headers, or other sources
- Need type hints for disambiguation
- Want consistency across all data access

## Examples

### Simple Form Data

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: createUser
run:
  validation:
    required:
      - name
      - email
    properties:
      name:
        type: string
      email:
        type: string
        format: email
  
  expr:
    - set('userName', input.name)
    - set('userEmail', input.email)
  
  sql:
    connectionName: main
    query: |
      INSERT INTO users (name, email) 
      VALUES ($1, $2)
    params:
      - get('userName')
      - get('userEmail')
```

### Nested Object Access

<div v-pre>

```yaml
run:
  expr:
    # Access nested properties
    - set('shippingCity', input.order.shippingAddress.city)
    - set('billingEmail', input.order.billing.email)
    - set('itemCount', len(input.order.items))
  
  chat:
    prompt: |
      Process order with {{ get('itemCount') }} items.
      Shipping to: {{ get('shippingCity') }}
```

</div>

### Array Processing

<div v-pre>

```yaml
run:
  expr:
    - set('items', input.items)
    - set('totalItems', len(get('items')))
    - set('firstItem', get('items')[0])
  
  chat:
    prompt: |
      Process {{ get('totalItems') }} items.
      First item: {{ get('firstItem') }}
```

</div>

### Conditional Based on Input

<div v-pre>

```yaml
run:
  expr:
    - set('hasItems', input.items != nil && len(input.items) > 0)
    - set('isPremium', input.user.tier == 'premium')
  
  skipCondition:
    - !get('hasItems')
  
  chat:
    prompt: |
      {{ get('isPremium') ? 'Premium user' : 'Standard user' }}:
      Process {{ len(input.items) }} items.
```

</div>

## Input Object vs Request Body

The `input` object is the same as `request.body`:

```yaml
run:
  expr:
    # These are equivalent:
    - set('name1', input.name)
    - set('name2', request.body.name)
    - set('name3', get('name'))
```

## Limitations

1. **Body data only** - `input` only contains request body data, not query parameters or headers
2. **No type hints** - Cannot specify data source like `get('key', 'param')`
3. **No fallback** - Returns nil if property doesn't exist (use `default()` for fallbacks)

## Best Practices

1. **Use for body data** - Perfect for POST/PUT request bodies
2. **Combine with validation** - Validate input structure before using
3. **Use default() for safety** - `default(input.field, 'fallback')`
4. **Prefer get() for flexibility** - When you need query params, headers, or type hints

## See Also

- [Unified API](unified-api.md) - Primary API for data access
- [Request Object](request-object.md) - Full request access
- [Validation](validation.md) - Input validation
