# Real-world workflows showcase

Each showcase agent is about 20 lines of YAML with no custom code. You POST a JSON body and get structured JSON back.

```text
POST /api/v1/run {"q": ...} -> chat resource -> {"success": true, "data": {...}}
```

## Read next

- [Full walkthrough](/reference/examples/showcase) - every agent, file and command
- [Examples](/examples/) - all examples
- [LLM resource](/workflow/resources/llm) - the chat resource
