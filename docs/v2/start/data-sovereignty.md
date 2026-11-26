---
title: Data Sovereignty
description: Run your own LLM and the workflow on servers you control, in the region you choose. Prompts, code, and data stay inside your boundary. No third-party AI provider is in the loop.
---

# Data sovereignty

With kdeps, you run the full AI stack on servers that you control, in the region that you select. The stack includes the model, the workflow, and the API in front of them. Your prompts, source code, and customer data stay in that boundary. They do not go to a third-party AI provider.

Point each workflow host at your own model server:

```yaml
# ~/.kdeps/config.yaml on each workflow host
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1   # your LLM server, your network
  models:
    - llama3.2                            # one model, or several
```

## Read next

- [Full text](/reference/start/data-sovereignty) - the complete page
- [Local models](/start/local-models) - llamafile and Ollama, fully offline
- [kdeps LLM server](/llm-server/) - run your own inference appliance
