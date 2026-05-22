# Items Reference

Use cases, dynamic items, collecting results, and best practices for the [`items:` field](/concepts/items).

## Use Cases

### Batch LLM Processing

<div v-pre>

```yaml
# resources/batch-queries.yaml
actionId: batchQueries
items:
  - "What is machine learning?"
  - "Explain neural networks"
  - "What is deep learning?"

chat:
  prompt: "{{ get('current') }}"
  jsonResponse: true
  jsonResponseKeys:
    - answer
```

</div>

### Data Enrichment

<div v-pre>

```yaml
# resources/enrich-products.yaml
actionId: enrichProducts
requires: [fetchProducts]
items: get('fetchProducts')

httpClient:
  method: GET
  url: "https://api.example.com/details/{{ get('current').id }}"
```

</div>

### Report Generation

<div v-pre>

```yaml
# resources/generate-report.yaml
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

chat:
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

<div v-pre>

```yaml
# resources/translate.yaml
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

chat:
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
# resources/chained-process.yaml
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

chat:
  prompt: |
    Step {{ get('current').step }}: {{ get('current').action }}

    {{ get('prev') ? 'Previous step output: ' + get('prev').result : 'Starting fresh.' }}

    Complete this step.
```

</div>

### Image Batch Processing

<div v-pre>

```yaml
# resources/analyze-images.yaml
actionId: analyzeImages
items:
  - path: "/uploads/image1.jpg"
  - path: "/uploads/image2.jpg"
  - path: "/uploads/image3.jpg"

chat:
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
actionId: processItems
items:
  - "Item 1"
  - "Item 2"

chat:
  prompt: "Process {{ get('current') }}"

---
# Response resource
actionId: response
requires: [processItems]
apiResponse:
  response:
    results: get('processItems')      # All results as array
    first: get('processItems')[0]     # First result
    count: len(get('processItems'))   # Result count
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

- Items process sequentially - each waits for the previous to complete
- Set `timeout` to account for total processing time across all items
- For large datasets, Python batch processing is more efficient than items

## Best Practices

- Use skip conditions to avoid processing items that don't qualify
- Use `prev` / `next` accessors to pass context between items
- For very large batches, use the `python:` resource with pandas

## See Also

- [Items](/concepts/items) - Core items concept and syntax
- [Expressions](/concepts/expressions) - Dynamic item expressions
- [Python Resource](/resources/python) - Batch processing alternative
