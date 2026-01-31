# Items Iteration

Items allow you to process multiple values in sequence, executing a resource for each item.

## Basic Usage

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: processItems

items:
  - "Item 1"
  - "Item 2"
  - "Item 3"

run:
  chat:
    model: llama3.2:1b
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
run:
  expr:
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: processWithItemObject
items:
  - "first"
  - "second"
  - "third"
run:
  expr:
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: collectResults
  requires:
    - processItems
run:
  expr:
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
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: collectResults
  requires:
    - processItems
run:
  expr:
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
items:
  - "apple"
  - "banana"
  - "cherry"

run:
  chat:
    prompt: |
      Item {{ get('index') + 1 }} of {{ get('count') }}: {{ get('current') }}
      Describe this fruit.
```

</div>

### With Context

<div v-pre>

```yaml
items:
  - "Introduction"
  - "Main Content"
  - "Conclusion"

run:
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
items:
  - "process"
  - "skip_this"
  - "process"

run:
  skipCondition:
    - get('current') == 'skip_this'

  chat:
    prompt: "Processing: {{ get('current') }}"
```

</div>

### Conditional Processing

<div v-pre>

```yaml
items:
  - value: "Task 1"
    priority: "high"
  - value: "Task 2"
    priority: "low"
  - value: "Task 3"
    priority: "high"

run:
  skipCondition:
    - get('current').priority != 'high'

  chat:
    prompt: "Handle high-priority task: {{ get('current').value }}"
```

</div>

## Use Cases

### Batch LLM Processing

Process multiple queries:

<div v-pre>

```yaml
metadata:
  actionId: batchQueries

items:
  - "What is machine learning?"
  - "Explain neural networks"
  - "What is deep learning?"

run:
  chat:
    model: llama3.2:1b
    prompt: "{{ get('current') }}"
    jsonResponse: true
    jsonResponseKeys:
      - answer
```

</div>

### Data Enrichment

Enrich a list of records:

<div v-pre>

```yaml
metadata:
  actionId: enrichProducts
  requires: [fetchProducts]

# Items could come from a previous resource
items: get('fetchProducts')

run:
  httpClient:
    method: GET
    url: "https://api.example.com/details/{{ get('current').id }}"
```

</div>

### Report Generation

Generate sections of a report:

<div v-pre>

```yaml
metadata:
  actionId: generateReport

items:
  - section: "executive_summary"
    title: "Executive Summary"
  - section: "analysis"
    title: "Market Analysis"
  - section: "recommendations"
    title: "Recommendations"
  - section: "conclusion"
    title: "Conclusion"

run:
  chat:
    model: llama3.2:1b
    prompt: |
      Generate the "{{ get('current').title }}" section of the report.
      Data: {{ get('reportData') }}

      {{ get('prev') ? 'Previous section: ' + get('prev').title : '' }}
    jsonResponse: true
    jsonResponseKeys:
      - content
      - key_points
```

</div>

### Multi-Language Translation

Translate text to multiple languages:

<div v-pre>

```yaml
metadata:
  actionId: translate

items:
  - code: "es"
    name: "Spanish"
  - code: "fr"
    name: "French"
  - code: "de"
    name: "German"
  - code: "ja"
    name: "Japanese"

run:
  chat:
    model: llama3.2:1b
    prompt: |
      Translate to {{ get('current').name }}:
      "{{ get('originalText') }}"
    jsonResponse: true
    jsonResponseKeys:
      - translation
      - language_code
```

</div>

### Sequential Processing

<div v-pre>

```yaml
metadata:
  actionId: chainedProcess

items:
  - step: 1
    action: "gather_requirements"
  - step: 2
    action: "design_solution"
  - step: 3
    action: "implement"
  - step: 4
    action: "test"

run:
  chat:
    prompt: |
      Step {{ get('current').step }}: {{ get('current').action }}

      {{ get('prev') ? 'Previous step output: ' + get('prev').result : 'Starting fresh.' }}

      Complete this step.
```

</div>

### Image Batch Processing

Process multiple images:

<div v-pre>

```yaml
metadata:
  actionId: analyzeImages

# Items from file upload or list
items:
  - path: "/uploads/image1.jpg"
  - path: "/uploads/image2.jpg"
  - path: "/uploads/image3.jpg"

run:
  chat:
    model: llama3.2-vision
    prompt: "Describe this image"
    files:
      - "{{ get('current').path }}"
```

</div>

## Collecting Results

Item results are collected into an array:

<div v-pre>

```yaml
# Processing resource
metadata:
  actionId: processItems

items:
  - "Item 1"
  - "Item 2"

run:
  chat:
    prompt: "Process {{ get('current') }}"

---
# Response resource
metadata:
  actionId: response
  requires: [processItems]

run:
  apiResponse:
    response:
      # All results as array
      results: get('processItems')

      # First result
      first: get('processItems')[0]

      # Result count
      count: len(get('processItems'))
```

</div>

## Dynamic Items

Items can come from expressions or previous resources:

```yaml
# From expression
items: split(get('csv_data'), ',')

# From previous resource
items: get('fetchItems')

# Filtered items
items: filter(get('allItems'), .status == 'active')
```

## Performance Considerations

1. **Items process sequentially** - Each item waits for the previous to complete
2. **Set appropriate timeouts** - Account for total processing time
3. **Limit item count** - Large lists can be slow
4. **Consider batching** - For very large datasets, use Python for batch processing

## Best Practices

1. **Use skip conditions** - Skip items that don't need processing
2. **Handle errors gracefully** - One failing item shouldn't break the batch
3. **Provide context** - Use prev/next for related processing
4. **Collect results appropriately** - Use the right data structure for results
5. **Consider alternatives** - For large batches, Python may be more efficient

## Next Steps

- [Resources Overview](../resources/overview) - Resource configuration
- [Expressions](expressions) - Expression syntax
- [Expression Functions Reference](expression-functions-reference) - Complete function reference
- [Python Resource](../resources/python) - For complex batch processing
