---
outline: deep
---

# Web Server Mode

Kdeps can be extended to be a full-stack AI application by serving both backend APIs (powered by open-source LLMs) and
frontend interfaces. The Web Server Mode enables hosting static frontends (e.g., React, Vue, HTML dashboards) or
reverse-proxying to dynamic web apps (e.g., Streamlit, Node.js, Django, Rails). This makes Kdeps ideal for building,
testing, and deploying self-contained AI apps with integrated UIs and APIs.


## Configuration Overview

The `WebServerMode` setting toggles the web server. When enabled, Kdeps can serve static files or proxy to a local web
application. Configurations are defined in the `WebServer` block, specifying host, port, trusted proxies, and
routing rules.


```apl
// Enables or disables the web server.
WebServerMode = false

// Web server configuration block.
WebServer {
    // IP address to bind the server.
    // "127.0.0.1" for localhost; "0.0.0.0" for all interfaces.
    hostIP = "127.0.0.1"

    // Port to listen on (1–65535). Defaults to 8080.
    portNum = 8080

    // Optional: Trusted proxy IPs or CIDR blocks.
    // Leave empty to trust all proxies (avoid in production).
    trustedProxies {}

    // Routing rules for static files or reverse proxying.
    routes {
        new {
            // HTTP path to serve, e.g., "/ui" or "/dashboard".
            path = "/ui"

            // Server type: "static" for files, "app" for proxying.
            serverType = "static"

            // For serverType="static": Directory to serve files from.
            // Relative to /data/ in the agent.
            // Example: "/agentY/2.0.0/web" maps to /data/agentY/2.0.0/web
            publicPath = "/agentY/2.0.0/web/"

            // For serverType="app": Local port of the web app.
            // Required for serverType="app".
            // appPort = 3000

            // Optional: Shell command to start the app, run in publicPath.
            // Example: "streamlit run app.py" or "npm start"
            // command = ""
        }
    }
}
```

## WebServerMode

Set `WebServerMode = true` to activate the web server. This enables:

- **Static File Serving**: Host HTML, CSS, JavaScript, or images (e.g., React, Vue, Svelte SPAs) for dashboards, documentation, or UIs, seamlessly integrated with Kdeps' backend APIs and open-source LLMs.
- **Reverse Proxying**: Forward requests to a local web server (e.g., Node.js, Streamlit, Django) for dynamic applications like admin panels or interactive dashboards.
- **CORS Support**: Configure Cross-Origin Resource Sharing for secure API access from external frontends, with customizable origins, methods, and headers.

Each `routes` entry can independently serve static files or proxy to an app, supporting flexible multi-path setups.

## Example Use Cases

| Server Type | Use Case | Description |
|-------------|---------------------------------------|--------------------------------------------------------------|
| `static` | Serve React/Vue SPA | Host a frontend UI for visualizing LLM outputs or dashboards. |
| `app` | Proxy to Streamlit | Run an interactive data exploration app alongside Kdeps APIs. |
| `static` | Serve documentation | Deliver HTML-based model docs or reports. |
| `app` | Proxy to Django admin | Host an admin panel for managing AI workflows. |

## Example: Static Frontend and Streamlit App

This configuration serves a static frontend and proxies to a Streamlit app:

```apl
APIServer {
    cors {
        allowedOrigins {
            "http://localhost:8080"
        }
        allowedMethods {
            "GET"
            "POST"
        }
        allowedHeaders {
            "Content-Type"
        }
        allowCredentials = true
    }
}

WebServerMode = true

WebServer {
    hostIP = "0.0.0.0"
    portNum = 8080
    trustedProxies { "192.168.1.0/24" }

    routes {
        new {
            path = "/dashboard"
            serverType = "static"
            publicPath = "/agentX/1.0.0/dashboard/"
        }
        new {
            path = "/app"
            serverType = "app"
            appPort = 8501
            command = "streamlit run app.py"
            publicPath = "/agentX/1.0.0/streamlit/"
        }
    }
}
```

This setup:
- Serves a static dashboard from `/data/agentX/1.0.0/dashboard/` at `http://<host>:8080/dashboard`.
- Proxies to a Streamlit app on port 8501 at `http://<host>:8080/app`, launched with `streamlit run app.py`.
- Allows CORS for API calls from the frontend at `http://localhost:8080`.

## Best Practices

- **Security**: Set `trustedProxies` and restrict `cors.allowedOrigins` in production.
- **Ports**: Avoid conflicts by checking `portNum` and `appPort` with `netstat` or `lsof`.
- **Static Files**: Ensure `publicPath` exists under `/data/` and includes an `index.html`.
- **App Commands**: Verify `command` works in `publicPath` to start the app.
- **Logging**: Enable debug logs to troubleshoot routing, proxy, or CORS issues.

## Troubleshooting

- **404 Errors (Static)**: Check if `publicPath` exists and contains `index.html`.
- **Connection Refused (App)**: Confirm the app runs on `appPort` and `command` is valid.
- **CORS Errors**: Verify `allowedOrigins` matches the frontend’s domain and port.
- **Proxy Issues**: Ensure `trustedProxies` includes the proxy IP.
- **Startup Failures**: Review logs for directory contents or misconfigured paths.
