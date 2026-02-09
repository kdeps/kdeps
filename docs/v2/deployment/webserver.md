# WebServer Mode

WebServer mode enables serving static files and reverse proxying to web applications alongside your AI agent API.

## Overview

KDeps can serve:
- **Static files** - HTML, CSS, JavaScript, images
- **Single Page Applications** - React, Vue, Angular builds
- **Reverse proxy** - Streamlit, Gradio, Django, Flask apps
- **WebSocket connections** - Real-time applications

## Basic Configuration

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: fullstack-agent
  version: "1.0.0"
  targetActionId: responseResource

settings:
  apiServerMode: true
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]

  webServerMode: true
  webServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: "/"
        serverType: "static"
        publicPath: "./public"
```

## Static File Serving

Serve static files from a directory:

```yaml
settings:
  webServerMode: true
  webServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    routes:
      - path: "/"
        serverType: "static"
        publicPath: "./public"  # Relative to workflow directory

      - path: "/docs"
        serverType: "static"
        publicPath: "./documentation"

      - path: "/assets"
        serverType: "static"
        publicPath: "./static/assets"
```

### Directory Structure

```
my-agent/
├── workflow.yaml
├── resources/
├── public/               # Served at /
│   ├── index.html
│   ├── styles.css
│   └── app.js
└── documentation/        # Served at /docs
    └── index.html
```

### SPA (Single Page Application)

For React, Vue, or Angular apps:

```yaml
routes:
  - path: "/"
    serverType: "static"
    publicPath: "./frontend/build"
```

## Reverse Proxy

Forward requests to backend applications:

```yaml
settings:
  webServerMode: true
  webServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    routes:
      - path: "/app"
        serverType: "app"
        publicPath: "./streamlit-app"
        appPort: 8501
        command: "streamlit run app.py"
```

### Streamlit Example

```yaml
routes:
  - path: "/dashboard"
    serverType: "app"
    publicPath: "./dashboard"
    appPort: 8501
    command: "streamlit run dashboard.py --server.port 8501 --server.headless true"
```

The Streamlit app:

```python
# dashboard/dashboard.py
import streamlit as st
import requests

st.title("AI Dashboard")

query = st.text_input("Ask a question:")
if st.button("Submit"):
    response = requests.post(
        "http://localhost:16395/api/v1/chat",
        json={"q": query}
    )
    st.write(response.json()["response"]["answer"])
```

### Gradio Example

```yaml
routes:
  - path: "/demo"
    serverType: "app"
    publicPath: "./gradio-app"
    appPort: 7860
    command: "python app.py"
```

```python
# gradio-app/app.py
import gradio as gr
import requests

def chat(message):
    response = requests.post(
        "http://localhost:16395/api/v1/chat",
        json={"q": message}
    )
    return response.json()["response"]["answer"]

demo = gr.Interface(fn=chat, inputs="text", outputs="text")
demo.launch(server_name="0.0.0.0", server_port=7860)
```

### Flask Example

```yaml
routes:
  - path: "/admin"
    serverType: "app"
    publicPath: "./admin-panel"
    appPort: 5000
    command: "python app.py"
```

```python
# admin-panel/app.py
from flask import Flask, render_template
app = Flask(__name__)

@app.route('/')
def index():
    return render_template('index.html')

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
```

## WebSocket Support

WebSocket connections are automatically proxied:

```yaml
routes:
  - path: "/ws-app"
    serverType: "app"
    appPort: 8000
    command: "python websocket_app.py"
```

```python
# websocket_app.py
import asyncio
import websockets

async def handler(websocket, path):
    async for message in websocket:
        response = f"Echo: {message}"
        await websocket.send(response)

start_server = websockets.serve(handler, "0.0.0.0", 8000)
asyncio.get_event_loop().run_until_complete(start_server)
asyncio.get_event_loop().run_forever()
```

## Multiple Routes

Combine static files and multiple apps:

```yaml
settings:
  webServerMode: true
  webServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    routes:
      # Landing page (static)
      - path: "/"
        serverType: "static"
        publicPath: "./landing"

      # React dashboard (static build)
      - path: "/dashboard"
        serverType: "static"
        publicPath: "./dashboard/build"

      # Streamlit analytics
      - path: "/analytics"
        serverType: "app"
        appPort: 8501
        command: "streamlit run analytics.py"

      # Gradio demo
      - path: "/demo"
        serverType: "app"
        appPort: 7860
        command: "python demo.py"
```

## Trusted Proxies

For deployments behind a load balancer:

```yaml
settings:
  webServerMode: true
  webServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    trustedProxies:
      - "10.0.0.0/8"
      - "172.16.0.0/12"
      - "192.168.0.0/16"
    routes:
      - path: "/"
        serverType: "static"
        publicPath: "./public"
```

## Full-Stack Example

Complete example with API, static frontend, and Streamlit:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: fullstack-ai-app
  version: "1.0.0"
  targetActionId: responseResource

settings:
  # API Server
  apiServerMode: true
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
      - path: /api/v1/analyze
        methods: [POST]
      - path: /health
        methods: [GET]
    cors:
      enableCors: true
      allowOrigins:
        - http://localhost:16395

  # Web Server
  webServerMode: true
  webServer:
    hostIp: "0.0.0.0"
    portNum: 16395
    routes:
      # React frontend
      - path: "/"
        serverType: "static"
        publicPath: "./frontend/dist"

      # Streamlit admin
      - path: "/admin"
        serverType: "app"
        publicPath: "./admin"
        appPort: 8501
        command: "streamlit run admin.py --server.port 8501"

  # Agent settings
  agentSettings:
    pythonVersion: "3.12"
    pythonPackages:
      - streamlit
      - requests
    models:
      - llama3.2:1b
```

## Docker Considerations

When deploying with Docker:

```dockerfile
# Expose both ports
EXPOSE 16395 16395

# Run both servers
CMD ["./entrypoint.sh"]
```

Docker Compose:

```yaml
services:
  myagent:
    image: kdeps-myagent:1.0.0
    ports:
      - "16395:16395"  # API
      - "16395:16395"  # Web
```

## Best Practices

1. **Use CORS for APIs** - Allow frontend to call API
2. **Separate concerns** - API on one port, web on another
3. **Set appropriate timeouts** - For long-running apps
4. **Use health checks** - Monitor all endpoints
5. **Secure in production** - Use HTTPS via reverse proxy

## Troubleshooting

### App Not Starting

Check the command and ensure dependencies are installed:

```yaml
agentSettings:
  pythonPackages:
    - streamlit
    - gradio
```

### Port Conflicts

Ensure unique ports for each app:

```yaml
routes:
  - path: "/app1"
    appPort: 8501
  - path: "/app2"
    appPort: 8502
```

### CORS Issues

Enable CORS on the API server:

```yaml
apiServer:
  cors:
    enableCors: true
    allowOrigins:
      - http://localhost:16395
    allowMethods:
      - GET
      - POST
    allowCredentials: true
```

## Next Steps

- [Docker Deployment](docker) - Package for production
- [Workflow Configuration](../configuration/workflow) - Full settings reference
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples) - Working examples
