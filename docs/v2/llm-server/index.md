# kdeps LLM server

kdeps LLM server builds a standalone OpenAI-compatible inference appliance, with no
kdeps agent or workflow. Build it one time (Docker, ISO, or Kubernetes). Each kdeps host on your
network connects to it over `/v1`.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1
```

## Read next

- [kdeps LLM server reference](/reference/llm-server/) - every key, option and detail
- [Data sovereignty](/start/data-sovereignty) - prompts stay on your network
- [Agent loop](/workflow/agent-loop/) - a chat that calls tools, on this server
