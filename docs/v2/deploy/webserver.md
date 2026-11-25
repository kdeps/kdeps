# WebServer mode

`webServer:` serves static files or proxies to a running subprocess alongside your API server. Use it to serve a React frontend, a Streamlit dashboard, or any other web app next to your agent API.

```yaml
# workflow.yaml
deploy:
  webServer:
    routes:
      - path: "/"
        serverType: static    # or app: proxy to a running process
        publicPath: ./public
```

## Read next

- [WebServer mode reference](/reference/deploy/webserver) - every key, option and detail
- [Docker deployment](/deploy/docker) - Package for production
- [Workflow configuration](/workflow/configuration) - Full settings reference
