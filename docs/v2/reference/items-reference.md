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
items:
  - "get('fetchProducts')"

httpClient:
  method: GET
  url: "https://api.example.com/details/{{ get('current') }}"
```

</div>

### Report Generation

<div v-pre>

```yaml
# resources/generate-report.yaml
actionId: generateReport
items:
  - "executive_summary|Executive Summary"
  - "analysis|Market Analysis"
  - "recommendations|Recommendations"
  - "conclusion|Conclusion"

chat:
  prompt: |
    Generate the "{{ split(get('current'), '|')[1] }}" section of the report.
    Data: {{ get('reportData') }}

    {{ get('prev') ? 'Previous section: ' + get('prev') : '' }}
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
  - "es|Spanish"
  - "fr|French"
  - "de|German"
  - "ja|Japanese"

chat:
  prompt: |
    Translate to {{ split(get('current'), '|')[1] }}:
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
  - "1|gather_requirements"
  - "2|design_solution"
  - "3|implement"
  - "4|test"

chat:
  prompt: |
    Step {{ split(get('current'), '|')[0] }}: {{ split(get('current'), '|')[1] }}

    {{ get('prev') ? 'Previous step output: ' + get('prev') : 'Starting fresh.' }}

    Complete this step.
```

</div>

### Image Batch Processing

<div v-pre>

```yaml
# resources/analyze-images.yaml
actionId: analyzeImages
items:
  - "/uploads/image1.jpg"
  - "/uploads/image2.jpg"
  - "/uploads/image3.jpg"

chat:
  prompt: "Describe this image"
  files:
    - "{{ get('current') }}"
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

Items can come from expressions or previous resources. The `items:` field is always a YAML list; a single expression that returns an array is expanded into multiple iterations:

```yaml
# From expression - evaluates to array, each element becomes one iteration
items:
  - "split(get('csv_data'), ',')"

# From previous resource output
items:
  - "get('fetchItems')"

# Filtered items
items:
  - "filter(get('allItems'), .status == 'active')"
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
