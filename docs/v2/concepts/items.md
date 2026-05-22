# Items Iteration

`items:` runs a resource once per entry in a list -- like a for-each loop, but each iteration is a full resource execution with its own output.

## Basic Usage

<div v-pre>

```yaml
# resources/process-items.yaml

actionId: processItems
items:
  - "Item 1"
  - "Item 2"
  - "Item 3"

chat:
  prompt: "Process: {{ get('current') }}"
```

</div>

## Item Context

When processing items, special getters are available:

| Getter | Description |
|--------|-------------|
| `get('current')` | Current item value |
| `get('prev')` | Previous item (null if first) |
| `get('next')` | Next item (null if last) |
| `get('index')` | Current index (0-based) |
| `get('count')` | Total number of items |
| `get('all')` | Array of all items |

## The `item` Object

You can also access item context through the `item` object with callable methods:

### Method Syntax

```yaml
# resources/example.yaml
after:
  # Method-style access
  - set('curr', item.current())
  - set('prev', item.prev())
  - set('next', item.next())
  - set('idx', item.index())
  - set('cnt', item.count())
  - set('all', item.values())
```

### Comparison: get() vs item.method()

| get() Style | item.method() Style | Description |
|-------------|---------------------|-------------|
| `get('current')` | `item.current()` | Current item |
| `get('prev')` | `item.prev()` | Previous item |
| `get('next')` | `item.next()` | Next item |
| `get('index')` | `item.index()` | Current index |
| `get('count')` | `item.count()` | Total items |
| `get('all')` | `item.values()` | All items array |

Both syntaxes are equivalent. Use whichever is more readable for your use case.

### Example: Using item Object

<div v-pre>

```yaml
# resources/process-with-item-object.yaml
actionId: processWithItemObject
items:
  - "first"
  - "second"
  - "third"
after:
  # Using item object methods
  - set('position', "Item " + string(item.index() + 1) + " of " + string(item.count()))
  - set('hasPrevious', item.prev() != nil)
  - set('hasNext', item.next() != nil)
chat:
  prompt: |
    {{ get('position') }}
    Current: {{ item.current() }}
    {{ get('hasPrevious') ? 'After: ' + item.prev() : 'First item' }}
```

</div>

## Accessing All Item Values

After processing, you can access all collected values from a resource that uses items:

### Using `get('resourceId', 'itemvalues')`

```yaml
# resources/collect-results.yaml
actionId: collectResults
requires:
  - processItems
after:
  # Get all collected values from the items iteration
  - set('allResults', get('processItems', 'itemvalues'))
  - set('resultCount', len(get('allResults')))
apiResponse:
  response:
    results: get('allResults')
    count: get('resultCount')
```

### Using `item.values(actionID)`

You can also use the `item.values()` method with an action ID to get all iteration values from a specific resource:

```yaml
# resources/collect-results.yaml
actionId: collectResults
requires:
  - processItems
after:
  # Get all values from processItems resource
  - set('allResults', item.values('processItems'))
  - set('resultCount', len(get('allResults')))

apiResponse:
  response:
    results: get('allResults')
    count: get('resultCount')
```

**Note:** `item.values()` without arguments returns all items for the current iteration context (equivalent to `item.values()` or `get('all')`). With an action ID, it returns all values from that specific resource's items iteration.

## Examples

### Simple Processing

<div v-pre>

```yaml
# resources/example.yaml
items:
  - "apple"
  - "banana"
  - "cherry"

chat:
  prompt: |
    Item {{ get('index') + 1 }} of {{ get('count') }}: {{ get('current') }}
    Describe this fruit.
```

</div>

### With Context

<div v-pre>

```yaml
# resources/example.yaml
items:
  - "Introduction"
  - "Main Content"
  - "Conclusion"

chat:
  prompt: |
    Write the {{ get('current') }} section.
    {{ get('prev') ? 'Previous section was: ' + get('prev') : 'This is the first section.' }}
    {{ get('next') ? 'Next section will be: ' + get('next') : 'This is the last section.' }}
```

</div>

### Skip Specific Items

<div v-pre>

```yaml
# resources/example.yaml
items:
  - "process"
  - "skip_this"
  - "process"

validations:
  skip:
  - get('current') == 'skip_this'

chat:
  prompt: "Processing: {{ get('current') }}"
```

</div>

### Conditional Processing

<div v-pre>

```yaml
# resources/example.yaml
items:
  - value: "Task 1"
    priority: "high"
  - value: "Task 2"
    priority: "low"
  - value: "Task 3"
    priority: "high"

validations:
  skip:
  - get('current').priority != 'high'

chat:
  prompt: "Handle high-priority task: {{ get('current').value }}"
```

</div>

## See Also

- [Items Reference](/reference/items-reference) - Use cases, dynamic items, performance, best practices
- [Resources Overview](../resources/overview) - Resource configuration
- [Expressions](/concepts/expressions) - Expression syntax
- [Expression Functions Reference](/reference/expression-functions-reference) - Complete function reference
- [Python Resource](../resources/python) - For complex batch processing
