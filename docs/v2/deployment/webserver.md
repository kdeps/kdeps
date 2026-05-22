# WebServer Mode

`webServer:` serves static files or proxies to a running subprocess alongside your API server. Use it to serve a React frontend, a Streamlit dashboard, or any other web app next to your agent API.

## How routing works

Requests hit a single port. kdeps matches the path prefix and dispatches to the right handler.

```d2
direction: down

A: "incoming request\nport 16395" {shape: oval}
B: path prefix {shape: diamond}
C: "apiServer -> workflow DAG"
D: JSON response {shape: oval}
E: "subprocess on appPort\nstarted on demand\nproxy to localhost:appPort"
F: proxied response {shape: oval}
G: static files from publicPath
H: file or 404 {shape: oval}

A -> B
B -> C: "/api/v1/..."
B -> E: "/app/..."
B -> G: "/..."
C -> D
E -> F
G -> H
```

`apiServer` and `webServer` share the same port. Path prefix determines which handler fires.

## Basic Configuration

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: fullstack-agent
  version: "1.0.0"
  targetActionId: responseResource

settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]

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
# workflow.yaml
settings:
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
# workflow.yaml
routes:
  - path: "/"
    serverType: "static"
    publicPath: "./frontend/build"
```

## Reverse Proxy

Forward requests to backend applications:

```yaml
# workflow.yaml
settings:
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
# workflow.yaml
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
# workflow.yaml
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
# workflow.yaml
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
# workflow.yaml
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
# workflow.yaml
settings:
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
# workflow.yaml
settings:
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

## See Also

- [Docker Deployment](docker) - Package for production
- [Workflow Configuration](../configuration/workflow) - Full settings reference
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples) - Working examples
