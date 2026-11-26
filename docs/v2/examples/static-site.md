# Serve a static site

Serve a folder of static files (HTML, CSS, JS, images)
over HTTP using web server mode. There are no resources and no LLM - kdeps acts
as a plain static file server driven by `workflow.yaml`.

The whole workflow is a web server route:

```yaml
# workflow.yaml
deploy:
  webServer:
    portNum: 16395
    routes:
      - path: "/"
        serverType: "static"
        publicPath: "./frontend/build"   # e.g. Vite or Create React App output
```

## Read next

- [Full walkthrough](/reference/examples/static-site) - every step, file and command
- [Web server mode](/deploy/webserver) - reverse proxy, Streamlit, Gradio, Flask
- [Workflow configuration](/workflow/configuration) - the full `deploy` block
