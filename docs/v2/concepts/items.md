# Items iteration

`items:` runs a resource once per entry in a list - like a for-each loop, but each iteration is a full resource execution with its own output. This is a workflow mode concept; it also applies when the LLM calls a workflow as a tool in agent mode.

## Basic usage

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

## Item context

When processing items, special getters are available:

| Getter | Description |
|--------|-------------|
| `get('current')` | Current item value |
| `get('prev')` | Previous item (null if first) |
| `get('next')` | Next item (null if last) |
| `get('index')` | Current index (0-based) |
| `get('count')` | Total number of items |
| `get('all')` | Array of all items |

## The `item` object

You can also access item context through the `item` object with callable methods:

### Method syntax

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

### Example: using item object

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

## Accessing all item values

After processing, you can access all collected values from a resource that uses items:

### Using `item.values(actionID)`

Pass an action ID to `item.values()` to get all iteration values from a specific resource:

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

**Note:** `item.values()` without arguments returns all items for the current iteration context (equivalent to `get('all')`). With an action ID, it returns all values from that specific resource's items iteration.

## Examples

### Simple processing

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

### With context

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

### Skip specific items

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

### Conditional processing

Items iterate over expressions that evaluate to strings. To filter, use `skip` on the value:

<div v-pre>

```yaml
# resources/example.yaml
items:
  - "Task 1|high"
  - "Task 2|low"
  - "Task 3|high"

validations:
  skip:
    - "!(get('current') contains '|high')"

chat:
  prompt: "Handle high-priority task: {{ split(get('current'), '|')[0] }}"
```

</div>

## Use cases

- Run the same LLM prompt over each row of a dataset (summarize each document,
  classify each ticket, translate each string).
- Generate a multi-section document one section at a time, using `get('prev')`
  and `get('next')` for continuity.
- Fan a fixed list of URLs, files, or IDs through an `httpClient:`, `scraper:`,
  or `sql:` resource and collect the results.

## See also

- [Items reference](/reference/items-reference) - dynamic items, performance, best practices
- [Resources Overview](../resources/overview) - Resource configuration
- [Expressions](/concepts/expressions) - Expression syntax
- [Expression Functions Reference](/reference/expression-functions-reference) - Complete function reference
- [Python Resource](../resources/python) - For complex batch processing
