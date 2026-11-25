# Function calling

With tools, an LLM can call other resources during a response. kdeps runs the resource and gives its output back to the model.

```yaml
# resources/chat.yaml
chat:
  tools:
    list:
      - name: calculate
        description: Do math        # the model reads this to decide when to call
        script: calcTool            # actionId of the resource to run
```

## Read next

- [Function calling reference](/reference/workflow/tools) - every key, option and detail
- [LLM resource](/workflow/resources/llm) - Full LLM configuration
- [LLM backends](/workflow/resources/llm-backends) - Streaming and backend options
