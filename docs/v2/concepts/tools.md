# Tools (Function Calling)

Tools enable LLMs to call other resources or scripts to perform actions like calculations, database lookups, or API calls.

## Overview

When you define tools in a chat resource, the LLM can automatically decide when to use them based on the user's prompt. This enables powerful agentic workflows.

<div v-pre>

```yaml
run:
  chat:
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
    tools:
      - name: calculate
        description: Perform mathematical calculations
        script: calcTool
        parameters:
          expression:
            type: string
            description: Math expression to evaluate
            required: true
```

</div>

## Basic Tool Definition

```yaml
tools:
  - name: tool_name           # Unique identifier
    description: What it does # Help the LLM decide when to use it
    script: resourceId        # Reference to another resource
    parameters:               # Input parameters
      param_name:
        type: string          # string, number, integer, boolean, object, array
        description: What this parameter is for
        required: true        # Is it required?
```

## Tool Types

### Resource-Based Tools

Tools that reference other KDeps resources:

<div v-pre>

```yaml
# The tool resource
metadata:
  actionId: calcTool
run:
  python:
    script: |
      import json
      import math
      expr = """{{ get('expression') }}"""
      result = eval(expr, {"__builtins__": {}, "math": math})
      print(json.dumps({"result": result}))

---
# The LLM that uses the tool
metadata:
  actionId: llmWithTools
run:
  chat:
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
    tools:
      - name: calculate
        description: Evaluate mathematical expressions
        script: calcTool
        parameters:
          expression:
            type: string
            description: "Math expression (e.g., '2 + 2', 'math.sqrt(16)')"
            required: true
```

</div>

### Multiple Tools

Define multiple tools for different capabilities:

<div v-pre>

```yaml
run:
  chat:
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
    tools:
      - name: calculate
        description: Perform math calculations
        script: calcTool
        parameters:
          expression:
            type: string
            required: true

      - name: search_database
        description: Search the product database
        script: dbSearchTool
        parameters:
          query:
            type: string
            description: Search query
            required: true
          category:
            type: string
            description: Product category filter
            required: false
          limit:
            type: integer
            description: Maximum results
            required: false

      - name: send_email
        description: Send an email notification
        script: emailTool
        parameters:
          to:
            type: string
            required: true
          subject:
            type: string
            required: true
          body:
            type: string
            required: true
```

</div>

## Parameter Types

| Type | Description | Example |
|------|-------------|---------|
| `string` | Text value | `"hello"` |
| `number` | Float/decimal | `3.14` |
| `integer` | Whole number | `42` |
| `boolean` | True/false | `true` |
| `object` | JSON object | `{"key": "value"}` |
| `array` | List of values | `[1, 2, 3]` |

## Tool Execution Flow

```
User Prompt
    ↓
LLM analyzes prompt
    ↓
LLM decides to call tool(s)
    ↓
KDeps executes tool resource
    ↓
Tool result returned to LLM
    ↓
LLM generates final response
```

## Examples

### Calculator Tool

<div v-pre>

```yaml
# Calculator resource
metadata:
  actionId: calcTool
run:
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
    timeoutDuration: 30s

---
# LLM with calculator
metadata:
  actionId: mathAssistant
run:
  chat:
    model: llama3.2:1b
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
metadata:
  actionId: dbSearchTool
run:
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
metadata:
  actionId: productAssistant
run:
  chat:
    model: llama3.2:1b
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
metadata:
  actionId: weatherTool
run:
  httpClient:
    method: GET
    url: "https://api.openweathermap.org/data/2.5/weather?q={{ get('city') }}&appid={{ get('OPENWEATHER_API_KEY', 'env') }}&units=metric"
    timeoutDuration: 30s

---
# LLM with weather
metadata:
  actionId: weatherAssistant
run:
  chat:
    model: llama3.2:1b
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
metadata:
  actionId: smartAgent
run:
  chat:
    model: llama3.2:1b
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

### 1. Write Clear Descriptions

```yaml
# Good - specific and helpful
description: |
  Search the product database.
  Returns: id, name, price, category.
  Use for finding products by name or description.

# Bad - vague
description: Search stuff
```

### 2. Define All Parameters

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

### 3. Handle Errors in Tools

```python
try:
    result = perform_action()
    print(json.dumps({"success": True, "data": result}))
except Exception as e:
    print(json.dumps({"success": False, "error": str(e)}))
```

### 4. Keep Tools Focused

Each tool should do one thing well. Create multiple simple tools rather than one complex tool.

### 5. Use Appropriate Models

Tool calling works best with larger, instruction-tuned models:
- `llama3.2` (good)
- `llama3.2:1b` (basic tool calling)
- `mistral` (good)
- GPT-4, Claude (excellent)

## Debugging Tools

Add logging to understand tool execution:

<div v-pre>

```yaml
metadata:
  actionId: debugTool
run:
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

## Next Steps

- [LLM Resource](../resources/llm) - Full LLM configuration
- [Python Resource](../resources/python) - Building tool scripts
- [Unified API](unified-api) - Data access in tools
