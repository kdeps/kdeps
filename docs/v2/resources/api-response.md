# API Response Resource

`apiResponse:` builds and returns the HTTP response sent back to the caller. It is always the last resource in the dependency chain -- the resource pointed to by `targetActionId` in `workflow.yaml`.

## Basic Usage

```yaml

actionId: responseResource
name: API Response
requires:
  - llmResource
apiResponse:
  success: true
  response:
    data: get('llmResource')
  headers:
    Content-Type: application/json
```

## Configuration Options

```yaml
apiResponse:
  success: true                # Boolean: request success status
  response:                    # Response body (any structure)
    field: value
    nested:
      key: value
  headers:
    Header-Name: value
  model: llama3.2:1b
  backend: ollama
```

## Response Structure

### Simple Response

```yaml
apiResponse:
  success: true
  response:
    message: "Hello, World!"
```

Output:
```json
{
  "success": true,
  "response": {
    "message": "Hello, World!"
  }
}
```

### Dynamic Response

```yaml
apiResponse:
  success: true
  response:
    query: get('q')
    answer: get('llmResource').answer
    timestamp: info('timestamp')
    request_id: info('requestId')
```

### Nested Structure

```yaml
apiResponse:
  success: true
  response:
    user:
      id: get('user_id')
      name: get('userResource').name
    data:
      items: get('dataResource')
      count: get('dataResource').length
    pagination:
      page: get('page', '1')
      limit: get('limit', '10')
```

## Custom Headers

Set response headers:

<div v-pre>

```yaml
apiResponse:
  success: true
  response:
    data: get('result')
  headers:
    Content-Type: application/json
    X-Request-Id: info('requestId')
    X-Processing-Time: "{{ get('processingTime') }}ms"
    Cache-Control: "max-age=3600"
```

</div>

## Adding Metadata

Include model and backend information in responses:

```yaml
apiResponse:
  success: true
  response:
    answer: get('llmResource')
  model: llama3.2:1b
  backend: ollama
```

**Automatic Metadata**: If an LLM resource was used in the workflow, model and backend information are automatically added to the response metadata (unless explicitly specified).

<div v-pre>

```yaml
# LLM resource used earlier
actionId: llmResource
chat:
  model: llama3.2:1b
  backend: ollama
  prompt: "{{ get('q') }}"

---
# Response automatically includes model/backend
actionId: responseResource
requires: [llmResource]
apiResponse:
  success: true
  response:
    answer: get('llmResource')
  # model and backend added automatically
```

</div>

Response format:
```json
{
  "success": true,
  "data": {
    "answer": "The response text..."
  },
  "meta": {
    "model": "llama3.2:1b",
    "backend": "ollama"
  }
}
```

### Manual Metadata

You can also manually specify metadata to override automatic values:

```yaml
apiResponse:
  success: true
  response:
    answer: get('llmResource')
  model: custom-model
  backend: custom-backend
  headers:
    X-Custom-Header: value
```

**Note**: Manual metadata takes precedence over automatic metadata.

## Error Responses

For error handling, use preflight checks or conditional responses:

### Preflight Validation

```yaml
validations:
  check:
    - get('user_id') != ''
  error:
    code: 400
    message: User ID is required

apiResponse:
  success: true
  response:
    user: get('userResource')
```

### Conditional Success

```yaml
apiResponse:
  success: get('operationResource').status == 'success'
  response:
    result: get('operationResource').data
    error: get('operationResource').error
```

## Examples

### Chat API Response

```yaml
actionId: chatResponse
requires: [llmResource]
apiResponse:
  success: true
  response:
    answer: get('llmResource').answer
    model: llama3.2:1b
    usage:
      prompt_tokens: get('llmResource').prompt_tokens
      completion_tokens: get('llmResource').completion_tokens
  headers:
    Content-Type: application/json
```

### File Upload Response

```yaml
actionId: uploadResponse
requires: [processFile]
apiResponse:
  success: true
  response:
    message: File processed successfully
    file:
      name: get('file', 'filepath')
      type: get('file', 'filetype')
      size: get('processFile').size
    result: get('processFile').analysis
  headers:
    Content-Type: application/json
```

### Paginated List Response

```yaml
actionId: listResponse
requires: [fetchItems]
apiResponse:
  success: true
  response:
    items: get('fetchItems')
    pagination:
      page: get('page', '1')
      limit: get('limit', '10')
      total: get('fetchItems').total
      has_more: get('fetchItems').has_more
  headers:
    Content-Type: application/json
    X-Total-Count: get('fetchItems').total
```

### Multi-Resource Response

```yaml
actionId: dashboardResponse
requires:
  - userResource
  - statsResource
  - notificationsResource
apiResponse:
  success: true
  response:
    user:
      id: get('userResource').id
      name: get('userResource').name
      email: get('userResource').email
    stats:
      views: get('statsResource').views
      engagement: get('statsResource').engagement
    notifications:
      unread: get('notificationsResource').unread
      items: get('notificationsResource').items
  headers:
    Content-Type: application/json
```

### Error Response Pattern

```yaml
# Successful case
actionId: successResponse
requires: [dataResource]
validations:
  skip:
  - get('dataResource').error != null

apiResponse:
  success: true
  response:
    data: get('dataResource').data

---
# Error case
actionId: errorResponse
requires: [dataResource]
validations:
  skip:
  - get('dataResource').error == null

apiResponse:
  success: false
  response:
    error:
      code: get('dataResource').error.code
      message: get('dataResource').error.message
```

## Response Transformation

Transform data before returning:

```yaml
actionId: transformedResponse
requires: [rawData]
after:
  - set('formatted', formatData(get('rawData')))

apiResponse:
  success: true
  response:
    data: get('formatted')
    original_count: get('rawData').length
    processed_count: get('formatted').length
```

## Best Practices

1. **Always set Content-Type** - Usually `application/json`
2. **Include request ID** - Helps with debugging
3. **Be consistent** - Use the same structure across endpoints
4. **Include metadata** - Model version, timing, etc.
5. **Handle errors gracefully** - Clear error messages

## See Also

- [Resources Overview](overview) -- all resource types
- [LLM Resource](llm) -- AI model integration
- [Unified API](../concepts/unified-api) -- data access patterns
