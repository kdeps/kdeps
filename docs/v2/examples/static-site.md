# Serve a static site

*Applies to workflow mode.*

## Overview

In this tutorial you serve a folder of static files (HTML, CSS, JS, images)
over HTTP using web server mode. There are no resources and no LLM - kdeps acts
as a plain static file server driven by `workflow.yaml`.

This tutorial is for developers who have completed the
[quickstart](/getting-started/quickstart). It assumes you know:

- Basic YAML
- Basic HTML

By the end you will be able to:

- Configure web server mode with a static route
- Serve a `public/` directory
- Understand when to use web server mode instead of the API server

## Background

kdeps has two server modes. The API server (`apiServer:`) routes JSON requests
to resources. Web server mode (`webServer:`) serves static files or proxies to
another process - useful for a frontend, a docs site, or a dashboard sitting in
front of an agent API.

## Before you start

- kdeps installed (`kdeps --version`).
- A working directory for the project.

## Step 1: create the project and content

```bash
mkdir static-site
cd static-site
mkdir public
```

Create `public/index.html`:

```html
<!doctype html>
<html>
  <head><title>Hello from kdeps</title></head>
  <body>
    <h1>It works</h1>
    <p>Served by kdeps web server mode.</p>
  </body>
</html>
```

## Step 2: configure web server mode

Create `workflow.yaml`:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: static-site
  version: "1.0.0"
  targetActionId: "none"       # no resource runs; the server just serves files

settings:
  webServer:
    portNum: 16395
    routes:
      - path: "/"
        serverType: "static"   # serve files from disk
        publicPath: "./public" # the directory to serve
```

`targetActionId: "none"` tells kdeps there is no resource graph to run.

## Step 3: validate and run

```bash
kdeps validate .
kdeps run .
```

Open `http://localhost:16395/` in a browser, or:

```bash
curl http://localhost:16395/
```

You get the contents of `public/index.html`. Any other file under `public/` is
served at its matching path.

## Step 4: serve a built frontend (optional)

The same route serves a compiled single-page app - point `publicPath` at the
build output:

```yaml
# workflow.yaml
settings:
  webServer:
    portNum: 16395
    routes:
      - path: "/"
        serverType: "static"
        publicPath: "./frontend/build"   # e.g. Vite or Create React App output
```

## Summary

You served a static directory by:

- Using `webServer:` instead of `apiServer:`
- Setting `serverType: static` and `publicPath`
- Setting `targetActionId: "none"` because no resource runs

## Next steps

- [Web server mode](/deployment/webserver) - reverse proxy, Streamlit, Gradio, Flask
- [Workflow configuration](/configuration/workflow) - the full `settings` block
- [Docker deployment](/deployment/docker) - ship the site as an image
- [TLS and HTTPS](/deployment/tls-https) - serve it over HTTPS
