# Function calling

With tools, an LLM can call other resources during a response. When the LLM decides that it needs a tool, kdeps runs the target resource. kdeps gives the result back to the LLM, and the LLM continues. The LLM sees only the output of the tool. It does not see the resource YAML.

<div v-pre>

```yaml
# resources/chat.yaml
chat:
  context:
    prompt: "{{ get('q') }}"
  tools:
    list:
      - name: calculate
        description: Perform mathematical calculations  # LLM uses this to decide when to call
        script: calcTool                                # actionId of the resource to run
        parameters:
          expression:
            type: string
            description: Math expression to evaluate
            required: true
```

</div>

## Read next

- [Function calling reference](/reference/workflow/tools) - tool definition, tool types, resource-based tools, external MCP tools, multiple tools, parameter types, tool execution flow, use cases
- [Function calling](/reference/tools) - examples, tool chaining, best practices, debugging
- [LLM resource](/workflow/resources/llm) - Full LLM configuration
- [LLM backends](/workflow/resources/llm-backends) - Streaming and backend options
- [Python resource](/workflow/resources/python) - Building tool scripts
- [Unified API](/workflow/data-access) - Data access in tools
