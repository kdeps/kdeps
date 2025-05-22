<p align="center">
  <img src="./docs/public/logo.png" width="500" />
</p>

Kdeps is an all-in-one AI framework for building Dockerized full-stack AI applications (FE and BE) that includes
open-source LLM models out-of-the-box.

## Key Features

Kdeps is loaded with powerful features to streamline AI app development:

<details>
  <summary>🧩 Low-code/no-code capabilities</summary>
  Build <a href="https://kdeps.com/getting-started/configuration/workflow.html">operational full-stack AI apps</a>, enabling accessible development for non-technical users and production-ready applications.

```pkl
// workflow.pkl
name = "ticketResolutionAgent"
description = "Automates customer support ticket resolution with LLM responses."
version = "1.0.0"
targetActionID = "responseResource"
settings {
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/ticket"; methods { "POST" } }
    }
    cors { enableCORS = true; allowOrigins { "http://localhost:8080" } }
  }
  agentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2:1b" }
    ollamaImageTag = "0.6.8"
  }
}
```

```pkl
// resources/fetch_data.pkl
actionID = "httpFetchResource"
name = "CRM Fetch"
description = "Fetches ticket data via CRM API."
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/ticket" }
  preflightCheck {
    validations { "@(request.data().ticket_id)" != "" }
  }
  HTTPClient {
    method = "GET"
    url = "https://crm.example.com/api/ticket/@(request.data().ticket_id)"
    headers { ["Authorization"] = "Bearer @(session.getItem('crm_token'))" }
    timeoutDuration = 30.s
  }
}
```

```pkl
// resources/llm.pkl
actionID = "llmResource"
name = "LLM Ticket Response"
description = "Generates responses for customer tickets."
requires { "httpFetchResource" }
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/ticket" }
  chat {
    model = "llama3.2:1b"
    role = "assistant"
    prompt = "Provide a professional response to the customer query: @(request.data().query)"
    scenario {
      new { role = "system"; prompt = "You are a customer support assistant. Be polite and concise." }
      new { role = "system"; prompt = "Ticket data: @(client.responseBody("httpFetchResource"))" }
    }
    JSONResponse = true
    JSONResponseKeys { "response_text" }
    timeoutDuration = 60.s
  }
}
```

```pkl
// resources/response.pkl
actionID = "responseResource"
name = "API Response"
description = "Returns ticket resolution response."
requires { "llmResource" }
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/ticket" }
  APIResponse {
    success = true
    response {
      data { "@(llm.response('llmResource'))" }
    }
    meta { headers { ["Content-Type"] = "application/json" } }
  }
}
```
</details>

<details>
  <summary>🐳 Dockerized full-stack AI apps</summary>
  Build applications with <a href="https://kdeps.com/getting-started/introduction/quickstart.html#quickstart">batteries included</a> for seamless development and deployment, as detailed in the <a href="https://kdeps.com/getting-started/configuration/workflow.html#ai-agent-settings">AI agent settings</a>.

```pkl
# Creating a Docker image of the kdeps AI agent is easy!
# First, package the AI agent project.
$ kdeps package tickets-ai/
INFO kdeps package created package-file=tickets-ai-1.0.0.kdeps
# Then build a docker image and run.
$ kdeps run tickets-ai-1.0.0.kdeps
# It also creates a Docker compose configuration file.
```

```pkl
# docker-compose.yml
version: '3.8'
services:
  kdeps-tickets-ai-cpu:
    image: kdeps-tickets-ai:1.0.0
    ports:
      - "127.0.0.1:3000"
    restart: on-failure
    volumes:
      - ollama:/root/.ollama
      - kdeps:/.kdeps
volumes:
  ollama:
    external:
      name: ollama
  kdeps:
    external:
      name: kdeps
```
</details>

<details>
  <summary>🖼️ Support for vision or multimodal LLMs</summary>
  Process text, images, and other data types in a single workflow with <a href="https://kdeps.com/getting-started/resources/multimodal.html">vision or multimodal LLMs</a>.


```pkl
// workflow.pkl
name = "visualTicketAnalyzer"
description = "Analyzes images in support tickets for defects using a vision model."
version = "1.0.0"
targetActionID = "responseResource"
settings {
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/visual-ticket"; methods { "POST" } }
    }
    cors { enableCORS = true; allowOrigins { "http://localhost:8080" } }
  }
  agentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2-vision" }
    ollamaImageTag = "0.6.8"
  }
}
```

```pkl
// resources/fetch_data.pkl
actionID = "httpFetchResource"
name = "CRM Fetch"
description = "Fetches ticket data via CRM API."
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/ticket" }
  preflightCheck {
    validations { "@(request.data().ticket_id)" != "" }
  }
  HTTPClient {
    method = "GET"
    url = "https://crm.example.com/api/ticket/@(request.data().ticket_id)"
    headers { ["Authorization"] = "Bearer @(session.getItem('crm_token'))" }
    timeoutDuration = 30.s
  }
}
```

```pkl
// resources/llm.pkl
actionID = "llmResource"
name = "Visual Defect Analyzer"
description = "Analyzes ticket images for defects."
requires { "httpFetchResource" }
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/visual-ticket" }
  preflightCheck {
    validations { "@(request.filecount())" > 0 }
  }
  chat {
    model = "llama3.2-vision"
    role = "assistant"
    prompt = "Analyze the image for product defects and describe any issues found."
    files { "@(request.files()[0])" }
    scenario {
      new { role = "system"; prompt = "You are a support assistant specializing in visual defect detection." }
      new { role = "system"; prompt = "Ticket data: @(client.responseBody("httpFetchResource"))" }
    }
    JSONResponse = true
    JSONResponseKeys { "defect_description"; "severity" }
    timeoutDuration = 60.s
  }
}
```

```pkl
// resources/response.pkl
actionID = "responseResource"
name = "API Response"
description = "Returns defect analysis result."
requires { "llmResource" }
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/visual-ticket" }
  APIResponse {
    success = true
    response {
      data { "@(llm.response('llmResource'))" }
    }
    meta { headers { ["Content-Type"] = "application/json" } }
  }
}
```
</details>

<details>
  <summary>🔌 Create custom AI APIs</summary>
  Serve <a href="https://kdeps.com/getting-started/configuration/workflow.html#llm-models">open-source LLMs</a> through custom <a href="https://kdeps.com/getting-started/configuration/workflow.html#api-server-settings">AI APIs</a> for robust AI-driven applications.
</details>

<details>
  <summary>🌐 Pair APIs with frontend apps</summary>
  Integrate with frontend apps like Streamlit, NodeJS, and more for interactive AI-driven user interfaces, as outlined in <a href="https://kdeps.com/getting-started/configuration/workflow.html#web-server-settings">web server settings</a>.

```pkl
// workflow.pkl
name = "frontendAIApp"
description = "Pairs an AI API with a Streamlit frontend for text summarization."
version = "1.0.0"
targetActionID = "responseResource"
settings {
  APIServerMode = true
  WebServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/summarize"; methods { "POST" } }
    }
  }
  WebServer {
    hostIP = "127.0.0.1"
    portNum = 8501
    routes {
      new {
        path = "/app"
        publicPath = "/fe/1.0.0/web/"
        serverType = "app"
        appPort = 8501
        command = "streamlit run app.py"
      }
    }
  }
  agentSettings {
    timezone = "Etc/UTC"
    pythonPackages { "streamlit" }
    models { "llama3.2:1b" }
    ollamaImageTag = "0.6.8"
  }
}
```

```pkl
// data/fe/web/app.py (Streamlit frontend)
import streamlit as st
import requests

st.title("Text Summarizer")
text = st.text_area("Enter text to summarize")
if st.button("Summarize"):
  response = requests.post("http://localhost:3000/api/v1/summarize", json={"text": text})
  if response.ok:
    st.write(response.json()['response']['data']['summary'])
  else:
    st.error("Error summarizing text")
```

```pkl
// resources/llm.pkl
actionID = "llmResource"
name = "Text Summarizer"
description = "Summarizes input text using an LLM."
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/summarize" }
  chat {
    model = "llama3.2:1b"
    role = "assistant"
    prompt = "Summarize this text in 50 words or less: @(request.data().text)"
    JSONResponse = true
    JSONResponseKeys { "summary" }
    timeoutDuration = 60.s
  }
}
```
</details>

<details>
  <summary>🛠️ Let LLMs run tools automatically (aka MCP or A2A)</summary>
  Enhance functionality through scripts and sequential tool pipelines with <a href="https://kdeps.com/getting-started/resources/llm.html#tools-configuration">external tools and chained tool workflows</a>.


```pkl
// workflow.pkl
name = "toolChainingAgent"
description = "Uses LLM to query a database and generate a report via tools."
version = "1.0.0"
targetActionID = "responseResource"
settings {
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/report"; methods { "POST" } }
    }
  }
  agentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2:1b" }
    ollamaImageTag = "0.6.8"
  }
}
```

```pkl
// resources/llm.pkl
actionID = "llmResource"
name = "Report Generator"
description = "Generates a report using a database query tool."
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/report" }
  chat {
    model = "llama3.2:1b"
    role = "assistant"
    prompt = "Generate a sales report based on database query results. Date range: @(request.params("date_range"))"
    tools {
      new {
        name = "query_sales_db"
        script = "@(data.filepath('tools/1.0.0', 'query_sales.py'))"
        description = "Queries the sales database for recent transactions"
        parameters {
          ["date_range"] { required = true; type = "string"; description = "Date range for query (e.g., '2025-01-01:2025-05-01')" }
        }
      }
    }
    JSONResponse = true
    JSONResponseKeys { "report" }
    timeoutDuration = 60.s
  }
}
```

```pkl
// data/tools/query_sales.py
import sqlite3
import sys

def query_sales(date_range):
  start, end = date_range.split(':')
  conn = sqlite3.connect('sales.db')
  cursor = conn.execute("SELECT * FROM transactions WHERE date BETWEEN ? AND ?", (start, end))
  results = cursor.fetchall()
  conn.close()
  return results

print(query_sales(sys.argv[1]))
```
</details>

## Additional Features

<details>
  <summary>📈 Context-aware RAG workflows</summary>
  Enable accurate, knowledge-intensive tasks with <a href="https://kdeps.com/getting-started/resources/kartographer.html">RAG workflows</a>.
</details>

<details>
  <summary>📊 Generate structured outputs</summary>
  Create consistent, machine-readable responses from LLMs, as described in the <a href="https://kdeps.com/getting-started/resources/llm.html#chat-block">chat block documentation</a>.

```pkl
// workflow.pkl
name = "structuredOutputAgent"
description = "Generates structured JSON responses from LLM."
version = "1.0.0"
targetActionID = "responseResource"
settings {
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/structured"; methods { "POST" } }
    }
  }
  agentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2:1b" }
    ollamaImageTag = "0.6.8"
  }
}
```

```pkl
// resources/llm.pkl
actionID = "llmResource"
name = "Structured Response Generator"
description = "Generates structured JSON output."
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/structured" }
  chat {
    model = "llama3.2:1b"
    role = "assistant"
    prompt = "Analyze this text and return a structured response: @(request.data().text)"
    JSONResponse = true
    JSONResponseKeys { "summary"; "keywords" }
    timeoutDuration = 60.s
  }
}
```
</details>

<details>
  <summary>🤖 Leverage multiple open-source LLMs</summary>
  Use LLMs from <a href="https://kdeps.com/getting-started/configuration/workflow.html#llm-models">Ollama</a> and <a href="https://github.com/kdeps/examples/tree/main/huggingface_imagegen_api">Huggingface</a> for diverse AI capabilities.


```pkl
// workflow.pkl
models {
  "tinydolphin"
  "llama3.3"
  "llama3.2-vision"
  "llama3.2:1b"
  "mistral"
  "gemma"
  "mistral"
}
```
</details>

<details>
  <summary>🗂️ Upload documents or files</summary>
  Process documents for LLM analysis, ideal for document analysis tasks, as shown in the <a href="https://kdeps.com/getting-started/tutorials/files.html">file upload tutorial</a>.


```pkl
// workflow.pkl
name = "docAnalysisAgent"
description = "Analyzes uploaded documents with LLM."
version = "1.0.0"
targetActionID = "responseResource"
settings {
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/doc-analyze"; methods { "POST" } }
    }
  }
  agentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2-vision" }
    ollamaImageTag = "0.6.8"
  }
}
```

```pkl
// resources/llm.pkl
actionID = "llmResource"
name = "Document Analyzer"
description = "Extracts text from uploaded documents."
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/doc-analyze" }
  preflightCheck {
    validations { "@(request.filecount())" > 0 }
  }
  chat {
    model = "llama3.2-vision"
    role = "assistant"
    prompt = "Extract key information from this document."
    files { "@(request.files()[0])" }
    JSONResponse = true
    JSONResponseKeys { "key_info" }
    timeoutDuration = 60.s
  }
}
```
</details>

<details>
  <summary>🔄 Reusable AI agents</summary>
  Create flexible workflows with <a href="https://kdeps.com/getting-started/resources/remix.html">reusable AI agents</a>.

```pkl
// workflow.pkl
name = "docAnalysisAgent"
description = "Analyzes uploaded documents with LLM."
version = "1.0.0"
targetActionID = "responseResource"
workflows { "@ticketResolutionAgent" }
settings {
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/doc-analyze"; methods { "POST" } }
    }
  }
  agentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2-vision" }
    ollamaImageTag = "0.6.8"
  }
}
```

```pkl
// resources/response.pkl
actionID = "responseResource"
name = "API Response"
description = "Returns defect analysis result."
requires {
  "llmResource"
  "@ticketResolutionAgent/llmResource:1.0.0"
}
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/doc-analyze" }
  APIResponse {
    success = true
    response {
      data {
        "@(llm.response("llmResource"))"
        "@(llm.response('@ticketResolutionAgent/llmResource:1.0.0'))"
      }
    }
    meta { headers { ["Content-Type"] = "application/json" } }
  }
}
```
</details>

<details>
  <summary>🐍 Execute Python in isolated environments</summary>
  Run Python code securely using <a href="https://kdeps.com/getting-started/resources/python.html">Anaconda</a> in isolated environments.

```pkl
// resources/python.pkl
actionID = "pythonResource"
name = "Data Formatter"
description = "Formats extracted data for storage."
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/scan-document" }
  python {
    script = """
import pandas as pd

def format_data(data):
  df = pd.DataFrame([data])
  return df.to_json()

print(format_data(@(llm.response('llmResource'))))
"""
    timeoutDuration = 60.s
  }
}
```
</details>

<details>
  <summary>🌍 Make API calls</summary>
  Perform API calls directly from configuration, as detailed in the <a href="https://kdeps.com/getting-started/resources/client.html">client documentation</a>.

```pkl
// resources/http_client.pkl
actionID = "httpResource"
name = "DMS Submission"
description = "Submits extracted data to document management system."
run {
  restrictToHTTPMethods { "POST" }
  restrictToRoutes { "/api/v1/scan-document" }
  HTTPClient {
    method = "POST"
    url = "https://dms.example.com/api/documents"
    data { "@(python.stdout('pythonResource'))" }
    headers { ["Authorization"] = "Bearer @(session.getItem('dms_token'))" }
    timeoutDuration = 30.s
  }
}
```
</details>

<details>
  <summary>🚀 Run in Lambda or API mode</summary>
  Operate in <a href="https://kdeps.com/getting-started/configuration/workflow.html#lambda-mode">Lambda mode</a> or <a href="https://kdeps.com/getting-started/configuration/workflow.html#api-server-settings">API mode</a> for flexible deployment.
</details>

<details>
  <summary>✅ Built-in validations and checks</summary>
  Utilize <a href="https://kdeps.com/getting-started/resources/api-request-validations.html#api-request-validations">API request validations</a>, <a href="https://kdeps.com/getting-started/resources/validations.html">custom validation checks</a>, and <a href="https://kdeps.com/getting-started/resources/skip.html">skip conditions</a> for robust workflows.


```pkl
restrictToHTTPMethods { "POST" }
restrictToRoutes { "/api/v1/scan-document" }
preflightCheck {
  validations { "@(request.filetype('document'))" == "image/jpeg" }
}
skipCondition { "@(request.data().query.length)" < 5 }
```
</details>

<details>
  <summary>📁 Serve static websites or reverse-proxied apps</summary>
  Host <a href="https://kdeps.com/getting-started/configuration/workflow.html#static-file-serving">static websites</a> or <a href="https://kdeps.com/getting-started/configuration/workflow.html#reverse-proxying">reverse-proxied apps</a> directly.

```pkl
// workflow.pkl
name = "frontendAIApp"
description = "Pairs an AI API with a Streamlit frontend for text summarization."
version = "1.0.0"
targetActionID = "responseResource"
settings {
  APIServerMode = true
  WebServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/summarize"; methods { "POST" } }
    }
  }
  WebServer {
    hostIP = "127.0.0.1"
    portNum = 8501
    routes {
      new {
        path = "/app"
        serverType = "app"
        appPort = 8501
        command = "streamlit run app.py"
      }
    }
  }
  agentSettings {
    timezone = "Etc/UTC"
    pythonPackages { "streamlit" }
    models { "llama3.2:1b" }
    ollamaImageTag = "0.6.8"
  }
}
```
</details>

<details>
  <summary>💾 Manage state with memory operations</summary>
  Store, retrieve, and clear persistent data using <a href="https://kdeps.com/getting-started/resources/memory.html">memory operations</a>.


```pkl
expr {
  "@(memory.setItem('user_data', request.data().data))"
}
local user_data = "@(memory.getItem('user_data'))"
```
</details>

<details>
  <summary>🔒 Configure CORS rules</summary>
  Set <a href="https://kdeps.com/getting-started/configuration/workflow.html#cors-configuration">CORS rules</a> directly in the workflow for secure API access.


```pkl
// workflow.pkl
cors {
  enableCORS = true
  allowOrigins { "https://example.com" }
  allowMethods { "GET"; "POST" }
}
```
</details>

<details>
  <summary>🛡️ Set trusted proxies</summary>
  Enhance API and frontend security with <a href="https://kdeps.com/getting-started/configuration/workflow.html#trustedproxies">trusted proxies</a>.


```pkl
// workflow.pkl
APIServerMode = true
APIServer {
  hostIP = "127.0.0.1"
  portNum = 3000
  routes {
    new { path = "/api/v1/proxy"; methods { "GET" } }
  }
  trustedProxies { "192.168.1.1"; "10.0.0.0/8" }
}
```
</details>

<details>
  <summary>🖥️ Run shell scripts</summary>
  Execute <a href="https://kdeps.com/getting-started/resources/exec.html">shell scripts</a> seamlessly within workflows.

```pkl
// resources/exec.pkl
actionID = "execResource"
name = "Shell Script Runner"
description = "Runs a shell script."
run {
  exec {
    command = """
echo "Processing request at $(date)"
"""
    timeoutDuration = 60.s
  }
}
```
</details>

<details>
  <summary>📦 Install Ubuntu packages</summary>
  Install <a href="https://kdeps.com/getting-started/configuration/workflow.html#ubuntu-packages">Ubuntu packages</a> via configuration for customized environments.

```pkl
// workflow.pkl
agentSettings {
  timezone = "Etc/UTC"
  packages {
    "tesseract-ocr"
    "poppler-utils"
    "npm"
    "ffmpeg"
  }
  ollamaImageTag = "0.6.8"
}
```
</details>

<details>
  <summary>📜 Define Ubuntu repositories or PPAs</summary>
  Configure <a href="https://kdeps.com/getting-started/configuration/workflow.html#ubuntu-repositories">Ubuntu repositories or PPAs</a> for additional package sources.

```pkl
// workflow.pkl
repositories {
  "ppa:alex-p/tesseract-ocr-devel"
}
```
</details>

<details>
  <summary>⚡ Written in high-performance Golang</summary>
  Benefit from the speed and efficiency of Golang for high-performance applications.
</details>

<details>
  <summary>📥 Easy to install</summary>
  Install and use Kdeps with a single command, as outlined in the <a href="https://kdeps.com/getting-started/introduction/installation.html">installation guide</a>.

```shell
# On macOS
brew install kdeps/tap/kdeps
# Windows, Linux, and macOS
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/refs/heads/main/install.sh | sh
```

</details>

## Getting Started

Ready to explore Kdeps? Install it with a single command: [Installation Guide](https://kdeps.com/getting-started/introduction/installation.html).

Check out practical [examples](https://github.com/kdeps/examples) to jumpstart your projects.
