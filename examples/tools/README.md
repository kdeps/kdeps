# LLM Tool Calling Example

This example demonstrates LLM tool/function calling in KDeps v2, allowing LLMs to declare available functions and potentially call them.

## Features

- ✅ Tool/function definitions in ChatConfig
- ✅ Tool parameter schemas (OpenAI Functions format)
- ✅ Tool calls returned by LLM
- ✅ Multiple tools per request

## Current Status

**Tool execution is now implemented!** ✅

- ✅ Tool definitions work
- ✅ LLM receives tool definitions
- ✅ LLM can return tool_calls in response
- ✅ Tool calls are automatically detected
- ✅ Tool resources are executed automatically
- ✅ Tool results are fed back to LLM
- ✅ Multi-turn conversations with tools supported

## Run Locally

```bash
# From examples/tools directory
kdeps run workflow.yaml --dev

# Or from root
kdeps run examples/tools/workflow.yaml --dev
```

## Test

### Query with Tools Available

```bash
curl -X POST 'http://localhost:16395/api/v1/tools?q=What%20is%20the%20weather%20in%20San%20Francisco?' \
  -H "Content-Type: application/json"
```

### Response

The LLM will receive the tool definitions and may return tool calls:

```json
{
  "success": true,
  "data": {
    "query": "What is the weather in San Francisco?",
    "llm_response": {
      "message": {
        "content": "I'll check the weather for you.",
        "tool_calls": [
          {
            "function": {
              "name": "get_weather",
              "arguments": "{\"location\": \"San Francisco, CA\"}"
            }
          }
        ]
      }
    },
    "tools_available": true
  }
}
```

## Structure

```
tools/
├── workflow.yaml              # Main workflow configuration
└── resources/
    ├── llm-with-tools.yaml   # LLM resource with tool definitions
    └── tool-response.yaml    # Response handler
```

## Key Concepts

### Tool Definition

Tools are defined in the `tools` field of ChatConfig:

```yaml
run:
  chat:
    model: "llama3.2:1b"
    prompt: "{{ get('q') }}"
    tools:
      - name: tool_name
        description: "Tool description"
        script: "resourceActionID"  # References a resource to execute
        parameters:
          param1:
            type: "string"
            description: "Parameter description"
            required: true
```

**Key Point**: The `script` field must reference a resource `actionID`. When the LLM calls the tool, that resource will be executed with the tool arguments available via `get()`.

### Tool Schema

Tools follow OpenAI Functions format:
- **name**: Function name
- **description**: What the tool does
- **parameters**: JSON Schema for parameters
  - **type**: Parameter type (string, number, boolean, object, array)
  - **description**: Parameter description
  - **required**: Whether parameter is required

### Supported Parameter Types

- `string` - Text values
- `number` - Numeric values
- `boolean` - True/false
- `object` - Complex objects (nested parameters)
- `array` - Lists of values

### Tool Call Response

When the LLM decides to use a tool, the response includes:
- `tool_calls` array with function name and arguments
- Original message content (often explaining what it will do)

## Example Tools

### 1. Weather Tool

```yaml
- name: get_weather
  description: "Get the current weather for a location"
  parameters:
    location:
      type: "string"
      description: "The city and state, e.g. San Francisco, CA"
      required: true
```

### 2. Calculator Tool

```yaml
- name: calculate
  description: "Perform a mathematical calculation. Supports basic arithmetic (+, -, *, /, **), math functions (sqrt, sin, cos, tan, log, exp, etc.), and constants (pi, e)"
  parameters:
    expression:
      type: "string"
      description: "Mathematical expression to evaluate (e.g. '2 + 2', 'sqrt(16)', 'sin(pi/2)')"
      required: true
```

### 3. Database Search Tool

```yaml
- name: search_database
  description: "Search the product database"
  parameters:
    query:
      type: "string"
      description: "Search query"
      required: true
    category:
      type: "string"
      description: "Product category filter"
      required: false
```

## Tool Execution Flow

Tool execution works as follows:

1. **LLM receives tool definitions** - Tools are sent to the LLM in the request
2. **LLM decides which tool to call** - Based on the prompt, LLM may return `tool_calls`
3. **Tool resources are executed** - Each tool call executes the referenced resource
4. **Tool arguments are available** - Tool parameters are stored in memory and accessible via `get()`
5. **Results are fed back to LLM** - Tool execution results are sent back to LLM
6. **LLM generates final response** - LLM processes tool results and generates final answer

### Multi-Turn Conversations

The system supports up to 5 iterations:
- LLM can call tools multiple times
- Tool results inform subsequent tool calls
- Final response includes all tool execution context

This enables powerful agent workflows where LLMs can:
- ✅ Query databases (via SQL resources)
- ✅ Call external APIs (via HTTP resources)
- ✅ Perform calculations (via Python/Exec resources)
- ✅ Execute Python scripts (via Python resources)
- ✅ Chain multiple tools together (automatic)

## Example: Calculator Tool

The `calculate` tool references the `calcTool` resource:

```yaml
# Tool definition
tools:
  - name: calculate
    script: "calcTool"  # Executes calcTool resource
    parameters:
      expression:
        type: "string"
        required: true

# calcTool resource (resources/calc-tool.yaml)
metadata:
  actionId: calcTool
run:
  python:
    script: |
      import json
      import math
      
      expression = "{{ get('expression', 'memory') }}"
      
      # Safely evaluate with restricted namespace
      allowed_names = {
          "__builtins__": {},
          'abs': abs, 'round': round, 'min': min, 'max': max,
          'sum': sum, 'pow': pow, 'int': int, 'float': float,
          'sqrt': math.sqrt, 'sin': math.sin, 'cos': math.cos, 'tan': math.tan,
          'log': math.log, 'log10': math.log10, 'exp': math.exp,
          'pi': math.pi, 'e': math.e, # ... and more
      }
      result = eval(expression, allowed_names, {})
      print(json.dumps({"result": result, "expression": expression}))
```

When LLM calls `calculate({"expression": "sqrt(16) + sin(pi/2)"})`:
1. `calcTool` resource is executed
2. `get('expression', 'memory')` returns the expression string
3. Python script safely evaluates the expression using restricted namespace
4. Result is returned as JSON
5. Result is fed back to LLM
6. LLM generates final response

**Supported operations:**
- Basic arithmetic: `+`, `-`, `*`, `/`, `**`, `%`
- Math functions: `sqrt()`, `sin()`, `cos()`, `tan()`, `log()`, `exp()`, `ceil()`, `floor()`, etc.
- Constants: `pi`, `e`
- Safe evaluation with restricted builtins to prevent code injection

## Notes

- Tool calling requires models that support function calling (check Ollama model capabilities)
- Tool `script` field must match a resource `actionID`
- Tool arguments are automatically stored in memory for resource access
- Maximum 5 tool call iterations to prevent infinite loops
- Tool resources can use any executor type (SQL, HTTP, Python, Exec, etc.)
