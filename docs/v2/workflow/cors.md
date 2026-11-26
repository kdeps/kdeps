# CORS configuration

CORS controls which browser origins can call your API. Set it inside the `cors:` block under `apiServer:`. If there is no `cors:` block, kdeps accepts all origins, with credentials enabled by default.

```yaml
# workflow.yaml
deploy:
  apiServer:
    cors:
      allowOrigins:
        - https://myapp.com
      allowCredentials: true
```

## Read next

- [CORS configuration reference](/reference/workflow/cors) - every key, option and detail
- [Workflow configuration](/workflow/configuration) - Full workflow settings reference
- [API server settings](/workflow/configuration#api-server-settings) - Complete API server configuration
