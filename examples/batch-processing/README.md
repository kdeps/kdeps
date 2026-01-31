# Batch Processing Example

This example demonstrates KDeps' items iteration feature for processing multiple items in parallel or sequentially.

## Overview

Items iteration allows you to process arrays of data efficiently:
- Process multiple API requests
- Transform collections of data
- Aggregate results from multiple sources
- Parallel or sequential execution

## Features Demonstrated

1. **Items Iteration** - Process array elements with `items: "{{...}}"`
2. **Conditional Skip** - Skip resources based on conditions
3. **Result Aggregation** - Combine results from multiple iterations
4. **Context Variables** - Access `{{item}}` in iterations

## Running the Example

```bash
# Start the server
kdeps run examples/batch-processing/workflow.yaml

# In another terminal, send a batch request
curl -X POST http://localhost:3000/process \
  -H "Content-Type: application/json" \
  -d @test-request.json
```

## Example Request

```json
{
  "items": [
    {"org": "kubernetes", "repo": "kubernetes"},
    {"org": "docker", "repo": "docker"},
    {"org": "golang", "repo": "go"}
  ]
}
```

The workflow will:
1. Fetch GitHub repo data for each item (parallel execution)
2. Transform each result to extract star counts
3. Aggregate all results into a final summary

## How Items Iteration Works

### 1. Declare Items Array

```yaml
resources:
  - id: process-item
    items: "{{input.items}}"  # Array from request
    config:
      # ... configuration that uses {{item}}
```

### 2. Access Current Item

Inside an items-iterating resource, use `{{item}}` to access the current element:

```yaml
url: "https://api.example.com/{{item.id}}"
```

### 3. Chain Iterations

You can iterate over previous results:

```yaml
resources:
  - id: first
    items: "{{input.data}}"
    # ... processes each item

  - id: second
    dependsOn: ["first"]
    items: "{{first}}"  # Iterate over first's results
    # ... processes each result from first
```

## Use Cases

### API Batch Requests
Process multiple API endpoints in parallel:

```yaml
items: "{{input.user_ids}}"
config:
  url: "https://api.example.com/users/{{item}}"
```

### File Processing
Process multiple files:

```yaml
items: "{{ctx.files}}"
config:
  command: "convert"
  args: ["{{item.path}}", "-resize", "800x600", "{{item.path}}.thumb.jpg"]
```

### Data Transformation
Transform array elements:

```yaml
items: "{{input.records}}"
config:
  prompt: "Summarize this record: {{item}}"
```

### Parallel LLM Calls
Process multiple prompts in parallel:

```yaml
items: "{{input.questions}}"
config:
  model: "llama3.2"
  prompt: "Answer this question: {{item}}"
```

## Advanced Patterns

### Conditional Processing

Skip items based on conditions:

```yaml
items: "{{input.items}}"
skip: "{{item.status == 'processed'}}"
```

### Nested Iterations

Process multidimensional data:

```yaml
# First level
- id: process-categories
  items: "{{input.categories}}"

  # Second level (processes each category's items)
  - id: process-items-in-category
    items: "{{item.items}}"
```

### Result Filtering

Filter results before aggregation:

```yaml
- id: aggregate
  dependsOn: ["process-items"]
  config:
    # Only aggregate successful results
    data: "{{filter(process-items, 'success == true')}}"
```

## Performance Notes

- Items are processed **in parallel** by default
- Use `dependsOn` to force sequential processing
- Large arrays are handled efficiently
- Results are collected automatically

## Next Steps

- See [complex-workflow](../complex-workflow/) for advanced dependencies
- See [http-advanced](../http-advanced/) for API request patterns
- Combine with LLM for batch AI processing
