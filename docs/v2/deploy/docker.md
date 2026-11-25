# Docker deployment

`kdeps bundle build` packages your workflow into a Docker image that starts an API server when run. kdeps generates the Dockerfile from your `workflow.yaml`.

```text
workflow.yaml -> kdeps bundle package -> .kdeps archive -> kdeps bundle build -> Docker image
```

```bash
kdeps bundle package path/to/workflow.yaml     # creates myagent-1.0.0.kdeps
kdeps bundle build myagent-1.0.0.kdeps         # creates image kdeps-myagent:1.0.0
```

`deploy.runtime` in `workflow.yaml` sets the base OS, Python packages and the LLM backend inside the image.

## Read next

- [Docker deployment reference](/reference/deploy/docker) - overview, packaging, building Docker images, GPU support, base OS auto-selection, LLM backend in images, ollama Docker images (opt-in), offline mode
- [Docker production reference](/reference/deploy/docker-reference) - production best practices, security hardening, troubleshooting
- [Workflow configuration](/workflow/configuration) - agent settings
- [Web server mode](/deploy/webserver) - serve frontends
- [LLM backends](/workflow/resources/llm-backends) - backend configuration
- [Management API](/reference/management-api) - live workflow updates with no rebuild
