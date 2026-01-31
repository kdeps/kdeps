# WebServer Static Files Example

This example demonstrates KDeps WebServer mode for serving static files.

## Overview

KDeps can serve static websites (HTML, CSS, JavaScript) directly without needing a separate web server like Nginx or Apache. This is perfect for:

- Serving dashboards and admin interfaces
- Hosting documentation sites
- Serving SPAs (Single Page Applications)
- Providing UI frontends for your AI workflows

## Features Demonstrated

1. **Static File Serving** - Serves files from the `./public` directory
2. **WebServer Mode** - Runs without API server or workflows
3. **Simple Configuration** - Just 3 settings in workflow.yaml

## Running the Example

```bash
# From the project root
kdeps run examples/webserver-static/workflow.yaml

# Or from this directory
kdeps run workflow.yaml
```

The server will start on http://127.0.0.1:8080

## Configuration

```yaml
settings:
  webServerMode: true  # Enable WebServer mode

  webServer:
    hostIp: "127.0.0.1"  # Bind to localhost
    portNum: 8080         # Port to listen on
    routes:
      - path: "/"          # URL path
        serverType: "static"  # Serve static files
        publicPath: "./public"  # Directory to serve from
```

## Directory Structure

```
webserver-static/
├── workflow.yaml     # KDeps configuration
├── public/           # Static files directory
│   └── index.html    # Homepage
└── README.md         # This file
```

## Use Cases

### Dashboard Hosting
Serve a React/Vue/Svelte dashboard that calls your AI APIs:

```yaml
routes:
  - path: "/"
    serverType: "static"
    publicPath: "./dashboard/build"
```

### Documentation Site
Host your API documentation:

```yaml
routes:
  - path: "/docs"
    serverType: "static"
    publicPath: "./docs/_site"
```

### Multiple Routes
Serve different static sites on different paths:

```yaml
routes:
  - path: "/"
    serverType: "static"
    publicPath: "./public"
  - path: "/admin"
    serverType: "static"
    publicPath: "./admin-ui/dist"
```

## Notes

- Files are served relative to the workflow.yaml location
- `index.html` is automatically served for directory requests
- All standard MIME types are supported
- No build step required - just static files

## Next Steps

- See [webserver-proxy](../webserver-proxy/) for app proxying examples
- Combine with API server mode for full-stack applications
- Deploy with Docker for production hosting
