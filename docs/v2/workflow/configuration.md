# Workflow configuration

`workflow.yaml` names the workflow and sets how it gets input and how it deploys. Resources are in `resources/`. Credentials are in `~/.kdeps/config.yaml`.

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: response   # the resource whose output is the result
deploy:
  apiServer:
    portNum: 16395
```

## Read next

- [Workflow configuration reference](/reference/workflow/configuration) - every key, option and detail
- [Resources overview](/workflow/resources) - Resource types and fields
- [Agencies](/agencies/) - Multi-agent orchestration
