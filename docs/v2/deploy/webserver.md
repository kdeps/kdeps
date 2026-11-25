# WebServer mode

`webServer:` serves static files or proxies to a running subprocess alongside your API server. Use it to serve a React frontend, a Streamlit dashboard, or any other web app next to your agent API.

```yaml
# workflow.yaml
deploy:
  webServer:
    rateLimit:
      requestsPerMinute: 120
      burst: 20
    maxBodyBytes: 1048576
    maxConcurrent: 50
    routes:
      - path: "/"
        serverType: "static"
        publicPath: "./public"
```

## Read next

- [WebServer mode reference](/reference/deploy/webserver) - how routing works, basic configuration, static file serving, reverse proxy, webSocket support, multiple routes, trusted proxies
- [Docker deployment](/deploy/docker) - Package for production
- [Workflow configuration](/workflow/configuration) - Full settings reference
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples) - Working examples
