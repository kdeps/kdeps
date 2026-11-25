# Docker deployment

`kdeps bundle build` packages your workflow into a Docker image that starts an API server when run. kdeps generates the Dockerfile from your `workflow.yaml`.

```text
workflow.yaml -> kdeps bundle package -> .kdeps archive -> kdeps bundle build -> Docker image
```

```bash
kdeps bundle package path/to/workflow.yaml     # creates myagent-1.0.0.kdeps
kdeps bundle build myagent-1.0.0.kdeps         # creates image kdeps-myagent:1.0.0
```

## Read next

- [Docker deployment reference](/reference/deploy/docker) - every key, option and detail
- [Workflow configuration](/workflow/configuration) - agent settings
- [Web server mode](/deploy/webserver) - serve frontends
