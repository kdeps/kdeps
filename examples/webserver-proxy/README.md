# WebServer Proxy Example

This example demonstrates KDeps WebServer mode with **reverse proxying** to a backend application.

## Overview

KDeps acts as a reverse proxy, forwarding requests from port 16395 to a Python backend running on port 8501. The backend app is automatically started by KDeps.

## Features Demonstrated

1. **Reverse Proxy** - Forward requests to backend application
2. **Automatic App Start** - KDeps launches the backend with `command`
3. **WebSocket Support** - Full WebSocket proxying (if backend uses it)
4. **Zero Configuration** - Backend app doesn't need to know about proxy

## Architecture

```
Client Request
      ↓
http://127.0.0.1:16395/ (KDeps WebServer)
      ↓ [Proxy]
http://127.0.0.1:8501/ (Python Backend)
      ↓
Response
```

## Running the Example

```bash
# Start the server (automatically starts backend)
kdeps run examples/webserver-proxy/workflow.yaml

# In another terminal, make requests
curl http://127.0.0.1:16395/
curl http://127.0.0.1:16395/api/data
curl http://127.0.0.1:16395/health
```

## How It Works

### 1. KDeps Configuration

```yaml
settings:
  webServerMode: true
  webServer:
    routes:
      - path: "/"
        serverType: "app"        # Proxy mode
        publicPath: "./backend"  # Working directory
        appPort: 8501            # Backend port
        command: "python3 server.py"  # Auto-start command
```

### 2. Backend Starts Automatically

KDeps executes `python3 server.py` in the `./backend` directory, starting the backend on port 8501.

### 3. Proxy Forwards Requests

All requests to `http://127.0.0.1:16395/*` are proxied to `http://127.0.0.1:8501/*`.

## Response Examples

### Root Endpoint
```bash
$ curl http://127.0.0.1:16395/
{
  "message": "Hello from Python backend!",
  "timestamp": "2026-01-12T02:10:13.657447",
  "status": "running",
  "proxied_by": "KDeps WebServer"
}
```

### API Endpoint
```bash
$ curl http://127.0.0.1:16395/api/data
{
  "items": [
    {"id": 1, "name": "Item 1"},
    {"id": 2, "name": "Item 2"},
    {"id": 3, "name": "Item 3"}
  ]
}
```

### Health Check
```bash
$ curl http://127.0.0.1:16395/health
{
  "status": "healthy"
}
```

## Use Cases

### Streamlit Applications
```yaml
routes:
  - path: "/app"
    serverType: "app"
    publicPath: "./streamlit-app"
    appPort: 8501
    command: "streamlit run app.py"
```

### React Development Server
```yaml
routes:
  - path: "/"
    serverType: "app"
    publicPath: "./frontend"
    appPort: 16395
    command: "npm start"
```

### Django Application
```yaml
routes:
  - path: "/api"
    serverType: "app"
    publicPath: "./django-app"
    appPort: 8000
    command: "python manage.py runserver 127.0.0.1:8000"
```

### Gradio Interface
```yaml
routes:
  - path: "/"
    serverType: "app"
    publicPath: "./gradio-app"
    appPort: 7860
    command: "python app.py"
```

## Multiple Routes

You can proxy to multiple backends:

```yaml
routes:
  # Frontend on root path
  - path: "/"
    serverType: "static"
    publicPath: "./dist"

  # API backend
  - path: "/api"
    serverType: "app"
    publicPath: "./backend"
    appPort: 8501
    command: "python3 api.py"

  # Admin interface
  - path: "/admin"
    serverType: "app"
    publicPath: "./admin"
    appPort: 8502
    command: "streamlit run admin.py --server.port 8502"
```

## Features

### Automatic Header Forwarding
All request headers are forwarded to the backend:
- `Host`
- `User-Agent`
- `Accept`
- Custom headers

### WebSocket Support
WebSocket connections are automatically detected and proxied:
```python
# Backend with WebSockets works automatically
import socketio

sio = socketio.Server()
app = socketio.WSGIApp(sio)
```

### Error Handling
If the backend is down, KDeps returns a 502 Bad Gateway error.

### Path Rewriting
Paths are automatically rewritten:
- Request: `http://127.0.0.1:16395/api/users`
- Forwarded to: `http://127.0.0.1:8501/api/users`

## Advantages

1. **Single Port** - Expose all services through one port
2. **No CORS Issues** - Frontend and backend on same origin
3. **Process Management** - KDeps starts/stops backend automatically
4. **Simple Deployment** - One command to run everything
5. **Development Mode** - Quick iteration with hot reload

## Troubleshooting

**Backend Not Starting:**
```bash
# Check if port is already in use
lsof -i :8501

# Check logs
tail -f /tmp/proxy-test.log
```

**Connection Refused:**
- Backend may not be listening on 127.0.0.1
- Check backend is binding to correct interface
- Verify port number matches

**502 Bad Gateway:**
- Backend crashed or exited
- Check backend logs for errors
- Verify command is correct

## Comparison with Nginx

### Nginx:
```nginx
server {
    listen 16395;
    location / {
        proxy_pass http://127.0.0.1:8501;
    }
}
```

### KDeps:
```yaml
routes:
  - path: "/"
    serverType: "app"
    appPort: 8501
    command: "python3 server.py"
```

**KDeps Advantages:**
- Starts backend automatically
- No separate config file
- Integrated with workflows
- Platform-independent

## Next Steps

- See [webserver-static](../webserver-static/) for serving static files
- Combine with API workflows for full-stack AI apps
- Add authentication middleware
- Deploy with Docker for production
