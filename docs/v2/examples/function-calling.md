# Give an LLM tools to call

Build an API where the LLM can call your own resources during a response. When
the model needs a calculation or a database lookup, it calls a tool. kdeps runs
the target resource and gives the result back. Then the model continues.

The chat lists a tool and the resource that runs it:

```yaml
# resources/chat.yaml (part)
chat:
  tools:
    list:
      - name: calculate
        description: "Evaluate a math expression."
        script: calcTool                 # the resource that runs this tool
        parameters:
          expression:
            type: string
            required: true
```

## Read next

- [Full walkthrough](/reference/examples/function-calling) - every step, file and command
- [Function calling](/workflow/tools) - MCP tools, multiple tools, parameter types
- [python component](/workflow/resources/python) - building tool scripts
