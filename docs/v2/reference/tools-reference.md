# Tools Reference

Examples, best practices, and debugging guidance for the [`tools:` block](/concepts/tools) in `chat:` resources.

## Examples

### Calculator Tool

<div v-pre>

```yaml
# Calculator resource
actionId: calcTool
python:
  script: |
    import json
    import math

    expression = """{{ get('expression') }}"""

    # Safe evaluation with math functions
    safe_dict = {
        'abs': abs, 'round': round,
        'min': min, 'max': max,
        'sum': sum, 'pow': pow,
        'sqrt': math.sqrt, 'sin': math.sin,
        'cos': math.cos, 'tan': math.tan,
        'log': math.log, 'log10': math.log10,
        'exp': math.exp, 'pi': math.pi, 'e': math.e
    }

    try:
        result = eval(expression, {"__builtins__": {}}, safe_dict)
        print(json.dumps({"result": result, "expression": expression}))
    except Exception as e:
        print(json.dumps({"error": str(e)}))
  timeout: 30s

---
# LLM with calculator
actionId: mathAssistant
chat:
  prompt: "{{ get('q') }}"
  tools:
    - name: calculate
      description: |
        Evaluate mathematical expressions.
        Supports: +, -, *, /, **, sqrt, sin, cos, tan, log, exp, pi, e
      script: calcTool
      parameters:
        expression:
          type: string
          description: "Math expression like '2 + 2' or 'sqrt(16)'"
          required: true
  jsonResponse: true
  jsonResponseKeys:
    - answer
    - calculation
```

</div>

### Database Search Tool

<div v-pre>

```yaml
# Database search resource
actionId: dbSearchTool
sql:
  connectionName: main
  query: |
    SELECT id, name, description, price, category
    FROM products
    WHERE (name ILIKE $1 OR description ILIKE $1)
      AND ($2 = '' OR category = $2)
    LIMIT $3
  params:
    - "'%' || get('query') || '%'"
    - get('category', '')
    - get('limit', '10')
  format: json

---
# LLM with search
actionId: productAssistant
chat:
  prompt: "{{ get('q') }}"
  tools:
    - name: search_products
      description: Search the product catalog
      script: dbSearchTool
      parameters:
        query:
          type: string
          description: Search terms
          required: true
        category:
          type: string
          description: Filter by category (electronics, clothing, etc.)
          required: false
        limit:
          type: integer
          description: Max results (default 10)
          required: false
```

</div>

### Weather API Tool

<div v-pre>

```yaml
# Weather API resource
actionId: weatherTool
httpClient:
  method: GET
  url: "https://api.openweathermap.org/data/2.5/weather?q={{ get('city') }}&appid={{ get('OPENWEATHER_API_KEY', 'env') }}&units=metric"
  timeout: 30s

---
# LLM with weather
actionId: weatherAssistant
chat:
  prompt: "{{ get('q') }}"
  tools:
    - name: get_weather
      description: Get current weather for a city
      script: weatherTool
      parameters:
        city:
          type: string
          description: City name (e.g., "London", "New York")
          required: true
```

</div>

### Multi-Tool Agent

<div v-pre>

```yaml
actionId: smartAgent
chat:
  prompt: "{{ get('q') }}"
  scenario:
    - role: system
      prompt: |
        You are a helpful assistant with access to tools.
        Use tools when needed to answer questions accurately.
        Always explain what you're doing.
  tools:
    - name: calculate
      description: Math calculations
      script: calcTool
      parameters:
        expression:
          type: string
          required: true

    - name: search_products
      description: Search product catalog
      script: dbSearchTool
      parameters:
        query:
          type: string
          required: true

    - name: get_weather
      description: Current weather
      script: weatherTool
      parameters:
        city:
          type: string
          required: true

    - name: send_notification
      description: Send a notification
      script: notifyTool
      parameters:
        message:
          type: string
          required: true
        channel:
          type: string
          description: "slack, email, or sms"
          required: true
```

</div>

## Tool Chaining

LLMs can chain multiple tools together:

**User**: "What's the total price of laptops under $1000 plus 8.5% tax?"

**LLM Flow**:
1. Calls `search_products(query="laptop", category="electronics")`
2. Gets results: `[{name: "Laptop A", price: 899}, {name: "Laptop B", price: 799}]`
3. Calls `calculate(expression="(899 + 799) * 1.085")`
4. Gets result: `1842.23`
5. Responds: "The total for laptops under $1000 with tax is $1842.23"

## Best Practices

### Write Clear Descriptions

```yaml
# Good - specific and helpful
description: |
  Search the product database.
  Returns: id, name, price, category.
  Use for finding products by name or description.

# Bad - vague
description: Search stuff
```

### Define All Parameters

```yaml
parameters:
  query:
    type: string
    description: "Search terms (e.g., 'wireless headphones')"
    required: true
  min_price:
    type: number
    description: "Minimum price filter"
    required: false
  max_price:
    type: number
    description: "Maximum price filter"
    required: false
```

### Handle Errors in Tools

```python
try:
    result = perform_action()
    print(json.dumps({"success": True, "data": result}))
except Exception as e:
    print(json.dumps({"success": False, "error": str(e)}))
```

### Keep Tools Focused

Each tool should do one thing well. Create multiple simple tools rather than one complex tool.

### Use Appropriate Models

Tool calling works best with larger, instruction-tuned models:
- `llama3.2` (good)
- `llama3.2:1b` (basic tool calling)
- `mistral` (good)
- GPT-4, Claude (excellent)

## Debugging Tools

Add logging to understand tool execution:

<div v-pre>

```yaml
actionId: debugTool
python:
  script: |
    import json
    import sys

    # Log to stderr (not captured as output)
    print(f"Tool called with: {{ get('params') }}", file=sys.stderr)

    # Process and return result
    result = {"status": "success"}
    print(json.dumps(result))
```

</div>

## See Also

- [Tools (Function Calling)](/concepts/tools) - Core tool definition and syntax
- [LLM Resource](/resources/llm) - Full LLM configuration
- [Python Resource](/resources/python) - Building tool scripts
- [Unified API](/concepts/unified-api) - Data access in tools
