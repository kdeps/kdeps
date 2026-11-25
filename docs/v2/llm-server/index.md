# kdeps LLM server

kdeps LLM server builds a standalone OpenAI-compatible inference appliance, with no
kdeps agent or workflow. Build it one time (Docker, ISO, or Kubernetes). Each kdeps host on your
network connects to it over `/v1`.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1
  # openai_api_key: "..."   # if appliance auth enabled
  models:
    - llama3.2              # optional allowlist
```

## Read next

- [kdeps LLM server reference](/reference/llm-server/index) - how it fits, commands, client contract, stock recipes, docker, kubernetes, ISO / bootable image, auth
- [Data sovereignty](/start/data-sovereignty) - prompts stay on your network
- [Agent loop](/workflow/agent-loop/) - a chat that calls tools, on this server
- [LLM backends](/workflow/resources/llm-backends) - client-side backend selection
- [LLM commands](/reference/llm-server/cli) - full CLI flag reference
- [Docker deployment](/deploy/docker) - agent images (not appliances)
- [Kubernetes deployment](/deploy/kubernetes) - agent manifests
- Example project: `examples/llm-server/`
