---
outline: deep
---

# Tools

The `tools` block lets open-source AI models (like LLaMA or Mistral) run scripts for tasks like math or file
operations. It supports Python (`.py`), TypeScript (`.ts`), JavaScript (`.js`), Ruby (`.rb`), or shell scripts (e.g.,
`.sh`), with inputs passed via `argv` (e.g., `sys.argv` in Python) or `$1`, `$2`, etc., for shell scripts. Scripts run
with: `.py` uses `python3`, `.ts` uses `ts-node`, `.js` uses `node`, `.rb` uses `ruby`, others use `sh`. The LLM can
automatically pick and chain multiple tools based on a prompt, using one tool’s output as input for the next. With
and `JSONResponseKeys`, tool outputs are structured as JSON for easier parsing. Tools are triggered via prompts or
manually with `@(tools.getItem(id))`, `runScript`, or `history`. This is like Anthropic’s MCP or Google’s A2A but for
open-source models only.


## What It Does

Inside a `chat` resource, the `tools` block lets the AI call scripts automatically via prompts or manually. The LLM can
chain tools, passing outputs as inputs, and with `JSONResponseKeys` structures results as JSON. It’s kdeps’ open-source
tool-calling system, similar to MCP or A2A but simpler.

## How It Looks

Create a `chat` resource:

```bash
kdeps scaffold [aiagent] llm
```

Define the `tools` block in the `chat` block. Here’s an excerpt:

```apl
chat {
    model = "llama3.2" // Open-source AI model
    role = "user"
    prompt = "Run the task using tools: @(request.params("q"))"
    JSONResponse = true
    JSONResponseKeys {
        "sum"      // Maps calculate_sum output to "result"
        "squared"  // Maps square_number output
        "saved"    // Maps write_result output
    }
    tools {
        new {
            name = "calculate_sum"
            script = "@(data.filepath("tools/1.0.0", "calculate_sum.py"))"
            description = "Add two numbers"
            parameters {
                ["a"] { required = true; type = "number"; description = "First number" }
                ["b"] { required = true; type = "number"; description = "Second number" }
            }
        }
        new {
            name = "square_number"
            script = "@(data.filepath("tools/1.0.0", "square_number.js"))"
            description = "Square a number"
            parameters {
                ["num"] { required = true; type = "number"; description = "Number to square" }
            }
        }
        new {
            name = "write_result"
            script = "@(data.filepath("tools/1.0.0", "write_result.sh"))"
            description = "Write a number to a file"
            parameters {
                ["path"] { required = true; type = "string"; description = "File path" }
                ["content"] { required = true; type = "string"; description = "Number to write" }
            }
        }
    }
    // Other settings like scenario, files, timeoutDuration...
}
```

## Sample Scripts

Stored in `tools/1.0.0/`:

### Python (calculate_sum.py)
Runs with `python3`, inputs via `sys.argv`.
```python
import sys
print(float(sys.argv[1]) + float(sys.argv[2]))
```

### JavaScript (square_number.js)
Runs with `node`, inputs via `process.argv`.
```javascript
const num = parseFloat(process.argv[2]);
console.log(num * num);
```

### Shell (write_result.sh)
Runs with `sh`, inputs via `$1`, `$2`.
```bash
echo "$2" > "$1"
```

## Key Pieces

- **new**: Defines a tool.
- **name**: Unique name, like `calculate_sum`.
- **script**: Script absolute path or using `@(data.filepath(...))`.
- **description**: Tool’s purpose.
- **parameters**:
  - **Key**: Parameter name, like `a`.
  - **required**: If needed.
  - **type**: Type, like `number` or `string`.
  - **description**: Parameter’s role.

## Schema Functions

- **getItem(id)**: Gets JSON output via `@(tools.getItem("id"))`. Returns text or empty string.
- **runScript(id, script, params)**: Runs a script with comma-separated parameters, returns JSON output.
- **history(id)**: Returns output history.

## Running Scripts

kdeps picks the program by file extension:
- `.py`: `python3`, inputs via `sys.argv`.
- `.ts`: `ts-node`, inputs via `process.argv`.
- `.js`: `node`, inputs via `process.argv`.
- `.rb`: `ruby`, inputs via `ARGV`.
- Others (e.g., `.sh`): `sh`, inputs as `$1`, `$2`, etc.

## Sample Prompts with Multi-Tool Chaining

The LLM selects and chains tools, structuring outputs as JSON. Prompts don’t name tools:

1. **Prompt**: “Add 6 and 4, square the result, and save it to ‘output.txt’.”
   - **Flow**:
     - LLM picks `calculate_sum` for 6 + 4 = 10.
     - Uses `square_number` for 10² = 100.
     - Calls `write_result` to save 100 to `output.txt`.
   - **JSON Output**:
     ```json
     {
       "result": 10,
       "squared_result": 100,
       "file_path": "output.txt"
     }
     ```

2. **Prompt**: “Sum 8 and 2, then write the sum to ‘sum.txt’.”
   - **Flow**:
     - LLM uses `calculate_sum` for 8 + 2 = 10.
     - Uses `write_result` to save 10 to `sum.txt`.
   - **JSON Output**:
     ```json
     {
       "result": 10,
       "file_path": "sum.txt"
     }
     ```

3. **Prompt**: “Add 5 and 5, square it twice, and save to ‘final.txt’.”
   - **Flow**:
     - LLM uses `calculate_sum` for 5 + 5 = 10.
     - Uses `square_number` for 10² = 100.
     - Uses `square_number` again for 100² = 10000.
     - Uses `write_result` to save 10000 to ‘final.txt’.
   - **JSON Output**:
     ```json
     {
       "result": 10,
       "squared_result": 10000,
       "file_path": "final.txt"
     }
     ```

## Manual Invocation

Run or get JSON results:
```apl
local result = "@(tools.runScript("square_number_123", "<path_to_script>", "10"))"
local output = "@(tools.getItem(\"square_number_123\"))"
```

## How It’s Like MCP or A2A

- **MCP**: Claude’s tool-calling, not supported in kdeps (open-source only).
- **A2A**: Google’s agent-connection system, unrelated to kdeps’ tool focus.
- **Kdeps**: Tool-calling with JSON outputs for open-source AI, like MCP but simpler.

## Tips

- Use unique `name` values.
- Write clear `description` fields for LLM tool selection.
- Define `JSONResponseKeys` for structured outputs.
- Check inputs with `required` and `type`.
- Secure scripts with `@(data.filepath(...))`.
- Set higher `timeoutDuration` in `chat` for longer tool chains.

## Open-Source Only

kdeps only supports open-source AI models, not Claude or MCP.

See [LLM Resource Functions](../resources/functions.md#llm-resource-functions) for more.
