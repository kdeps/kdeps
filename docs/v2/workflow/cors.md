# CORS configuration

CORS controls which browser origins can call your API. Set it inside the `cors:` block under `apiServer:`. If there is no `cors:` block, kdeps accepts all origins, with credentials enabled by default.

```yaml
# workflow.yaml
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
    cors:
      allowOrigins:
        - http://localhost:16395
        - https://myapp.com
      allowMethods:
        - GET
        - POST
        - OPTIONS
      allowHeaders:
        - Content-Type
        - Authorization
      allowCredentials: true
      maxAge: "24h"
```

## Read next

- [CORS configuration reference](/reference/workflow/cors) - basic configuration, configuration fields, common scenarios, how CORS works, best practices, troubleshooting, example: full configuration
- [Workflow configuration](/workflow/configuration) - Full workflow settings reference
- [API server settings](/workflow/configuration#api-server-settings) - Complete API server configuration
- [WebServer mode](/deploy/webserver) - Serving static files and proxying
