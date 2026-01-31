# Expressions

Expressions allow you to add logic, data transformation, and dynamic values to your workflow. They are powered by the [expr-lang](https://expr-lang.org/) engine.

## Where to Use Expressions

### 1. String Interpolation
Use <span v-pre>`{{ }}`</span> to embed expressions in any string field.

<div v-pre>

```yaml
run:
  chat:
    prompt: "Hello {{ get('name') }}, today is {{ info('timestamp') }}"
```

</div>

### 2. Expression Blocks (`expr`)
Use `expr` blocks to run logic steps. These are executed sequentially.

```yaml
run:
  expr:
    # Variable assignment
    - set('normalized_query', lower(trim(get('q'))))
    
    # Conditional logic
    - set('is_admin', get('role') == 'admin')
    
    # Math
    - set('total', get('price') * get('quantity'))
```

### 3. Conditions
Used in `skipCondition`, `preflightCheck`, and `onError`.

```yaml
skipCondition:
  - get('q') == ''

preflightCheck:
  validations:
    - len(get('password')) >= 8
```

## Expression Types

KDeps supports three types of expressions:

### Literal
Raw values returned as-is.
```yaml
key: "value"
count: 10
```

### Direct
Evaluated directly (used in `expr` blocks and conditions).
```yaml
- get('count') + 1
- user.name == 'Alice'
```

### Interpolated
String templates containing <span v-pre>`{{ }}`</span>.
<div v-pre>

```yaml
message: "Count is {{ get('count') }}"
```

</div>

## Standard Library

Since KDeps uses `expr-lang`, you have access to a rich standard library of functions:

- **String**: `trim()`, `lower()`, `upper()`, `split()`, `replace()`, `join()`
- **Math**: `min()`, `max()`, `abs()`, `ceil()`, `floor()`
- **List**: `len()`, `filter()`, `map()`, `first()`, `last()`, `contains()`
- **Type**: `int()`, `float()`, `string()`, `bool()`

See the [Expr Language Documentation](https://expr-lang.org/docs/language-definition) for a full list of built-in operators and functions.

## Helper Functions

KDeps adds specific helper functions for workflow context. See the [Expression Functions Reference](expression-functions-reference) for details.