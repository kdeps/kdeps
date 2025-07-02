<p align="center">
  <img src="./docs/public/logo.png" width="500" />
</p>

![version](https://img.shields.io/github/v/tag/kdeps/kdeps?style=flat-square&label=version)
![license](https://img.shields.io/github/license/kdeps/kdeps?style=flat-square)
![build](https://img.shields.io/github/actions/workflow/status/kdeps/kdeps/build-test.yml?branch=main&style=flat-square)
[![Go Report Card](https://goreportcard.com/badge/github.com/kdeps/kdeps)](https://goreportcard.com/report/github.com/kdeps/kdeps)
[![tests](https://img.shields.io/endpoint?style=flat-square&url=https://gist.githubusercontent.com/jjuliano/ce695f832cd51d014ae6d37353311c59/raw/kdeps-go-tests.json)](https://github.com/kdeps/kdeps/actions/workflows/build-test.yml)
[![coverage](https://img.shields.io/endpoint?style=flat-square&url=https://gist.githubusercontent.com/jjuliano/ce695f832cd51d014ae6d37353311c59/raw/kdeps-go-coverage.json)](https://github.com/kdeps/kdeps/actions/workflows/build-test.yml)

Kdeps is an all-in-one AI framework for building Dockerized full-stack AI applications (FE and BE) that includes
open-source LLM models out-of-the-box.

## 🆕 What's New in v0.3.1

- **Enhanced Schema Structure**: All PKL properties now use capitalized naming (e.g., `AgentID`, `Settings`, `ActionID`)
- **Improved Rate Limiting**: New `RateLimitMax` property for API throttling control
- **Environment Configuration**: Added `Environment` property for build environment selection
- **Enhanced Validations**: New `PostflightCheck`, `AllowedHeaders`, and `AllowedParams` properties
- **Retry Mechanisms**: Built-in `Retry` and `RetryTimes` properties for robust error handling
- **Better Resource Categorization**: Added `Name` and `Category` properties for resource organization
- **Simplified HTTP Methods**: API routes now use single `Method` string instead of arrays
- **Schema Assets**: Embedded schema files for improved reliability and offline development

## Key Features

Kdeps is loaded with features to streamline full-stack AI app development:

<details>
  <summary>🧩 Low-code/no-code capabilities</summary>
  Build <a href="https://kdeps.com/getting-started/configuration/workflow.html">operational full-stack AI apps</a>, enabling accessible development for non-technical users.

```pkl
// workflow.pkl
AgentID = "ticketResolutionAgent"
Description = "Automates customer support ticket resolution with LLM responses."
Version = "1.0.0"
TargetActionID = "responseResource"
Settings {
  RateLimitMax = 100
  Environment = "production"
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/ticket"; Method = "POST" }
    }
    cors { enableCORS = true; allowOrigins { "http://localhost:8080" } }
  }
  AgentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2:1b" }
    OllamaVersion = "0.8.0"
  }
}
```

```pkl
// resources/fetch_data.pkl
ActionID = "httpFetchResource"
Name = "CRM Fetch"
Description = "Fetches ticket data via CRM API."
Category = "integration"
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/ticket" }
  AllowedHeaders { "Authorization"; "Content-Type" }
  AllowedParams { "ticket_id" }
  PreflightCheck {
    Validations { "@(request.data().ticket_id)" != "" }
    Retry = true
    RetryTimes = 3
  }
  HTTPClient {
    Method = "GET"
    url = "https://crm.example.com/api/ticket/@(request.data().ticket_id)"
    headers { ["Authorization"] = "Bearer @(session.getRecord('crm_token'))" }
    TimeoutDuration = 30.s
  }
}
```

```pkl
// resources/llm.pkl
ActionID = "llmResource"
Name = "LLM Ticket Response"
Description = "Generates responses for customer tickets."
Category = "ai"
Requires { "httpFetchResource" }
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/ticket" }
  PreflightCheck {
    Validations { "@(request.data().query)" != "" }
    Retry = false
    RetryTimes = 1
  }
  PostflightCheck {
    Validations { "@(llm.response('llmResource').response_text)" != "" }
    Retry = true
    RetryTimes = 2
  }
  Chat {
    model = "llama3.2:1b"
    role = "assistant"
    prompt = "Provide a professional response to the customer query: @(request.data().query)"
    scenario {
      new { role = "system"; prompt = "You are a customer support assistant. Be polite and concise." }
      new { role = "system"; prompt = "Ticket data: @(client.responseBody("httpFetchResource"))" }
    }
    JSONResponse = true
    JSONResponseKeys { "response_text" }
    TimeoutDuration = 60.s
  }
}
```

```pkl
// resources/response.pkl
ActionID = "responseResource"
Name = "API Response"
Description = "Returns ticket resolution response."
Category = "output"
Requires { "llmResource" }
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/ticket" }
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
AgentID = "visualTicketAnalyzer"
Description = "Analyzes images in support tickets for defects using a vision model."
Version = "1.0.0"
TargetActionID = "responseResource"
Settings {
  RateLimitMax = 50
  Environment = "production"
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/visual-ticket"; Method = "POST" }
    }
    cors { enableCORS = true; allowOrigins { "http://localhost:8080" } }
  }
  AgentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2-vision" }
    OllamaVersion = "0.8.0"
  }
}
```

```pkl
// resources/fetch_data.pkl
ActionID = "httpFetchResource"
Name = "CRM Fetch"
Description = "Fetches ticket data via CRM API."
Category = "integration"
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/visual-ticket" }
  AllowedParams { "ticket_id" }
  PreflightCheck {
    Validations { "@(request.data().ticket_id)" != "" }
    Retry = true
    RetryTimes = 3
  }
  HTTPClient {
    Method = "GET"
    url = "https://crm.example.com/api/ticket/@(request.data().ticket_id)"
    headers { ["Authorization"] = "Bearer @(session.getRecord('crm_token'))" }
    TimeoutDuration = 30.s
  }
}
```

```pkl
// resources/llm.pkl
ActionID = "llmResource"
Name = "Visual Defect Analyzer"
Description = "Analyzes ticket images for defects."
Category = "ai"
Requires { "httpFetchResource" }
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/visual-ticket" }
  AllowedHeaders { "Content-Type" }
  PreflightCheck {
    Validations { "@(request.filecount())" > 0 }
    Retry = false
    RetryTimes = 1
  }
  PostflightCheck {
    Validations { "@(llm.response('llmResource').defect_description)" != "" }
    Retry = true
    RetryTimes = 2
  }
  Chat {
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
    TimeoutDuration = 60.s
  }
}
```

```pkl
// resources/response.pkl
ActionID = "responseResource"
Name = "API Response"
Description = "Returns defect analysis result."
Category = "output"
Requires { "llmResource" }
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/visual-ticket" }
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
AgentID = "frontendAIApp"
Description = "Pairs an AI API with a Streamlit frontend for text summarization."
Version = "1.0.0"
TargetActionID = "responseResource"
Settings {
  RateLimitMax = 200
  Environment = "development"
  APIServerMode = true
  WebServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/summarize"; Method = "POST" }
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
  AgentSettings {
    timezone = "Etc/UTC"
    pythonPackages { "streamlit" }
    models { "llama3.2:1b" }
    OllamaVersion = "0.8.0"
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
ActionID = "llmResource"
Name = "Text Summarizer"
Description = "Summarizes input text using an LLM."
Category = "ai"
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/summarize" }
  AllowedParams { "text" }
  PreflightCheck {
    Validations { "@(request.data().text)" != "" }
    Retry = false
    RetryTimes = 1
  }
  Chat {
    model = "llama3.2:1b"
    role = "assistant"
    prompt = "Summarize this text in 50 words or less: @(request.data().text)"
    JSONResponse = true
    JSONResponseKeys { "summary" }
    TimeoutDuration = 60.s
  }
}
```
</details>

<details>
  <summary>🛠️ Let LLMs run tools automatically (aka MCP or A2A)</summary>
  Enhance functionality through scripts and sequential tool pipelines with <a href="https://kdeps.com/getting-started/resources/llm.html#tools-configuration">external tools and chained tool workflows</a>.

```pkl
// workflow.pkl
AgentID = "toolChainingAgent"
Description = "Uses LLM to query a database and generate a report via tools."
Version = "1.0.0"
TargetActionID = "responseResource"
Settings {
  RateLimitMax = 50
  Environment = "production"
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/report"; Method = "POST" }
    }
  }
  AgentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2:1b" }
    OllamaVersion = "0.8.0"
  }
}
```

```pkl
// resources/llm.pkl
ActionID = "llmResource"
Name = "Report Generator"
Description = "Generates a report using a database query tool."
Category = "analytics"
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/report" }
  AllowedParams { "date_range" }
  PreflightCheck {
    Validations { "@(request.params("date_range"))" != "" }
    Retry = false
    RetryTimes = 1
  }
  Chat {
    model = "llama3.2:1b"
    role = "assistant"
    prompt = "Generate a sales report based on database query results. Date range: @(request.params("date_range"))"
    tools {
      new {
        Name = "query_sales_db"
        script = "@(data.filepath('tools/1.0.0', 'query_sales.py'))"
        Description = "Queries the sales database for recent transactions"
        parameters {
          ["date_range"] { required = true; type = "string"; Description = "Date range for query (e.g., '2025-01-01:2025-05-01')" }
        }
      }
    }
    JSONResponse = true
    JSONResponseKeys { "report" }
    TimeoutDuration = 60.s
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
AgentID = "structuredOutputAgent"
Description = "Generates structured JSON responses from LLM."
Version = "1.0.0"
TargetActionID = "responseResource"
Settings {
  RateLimitMax = 100
  Environment = "production"
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/structured"; Method = "POST" }
    }
  }
  AgentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2:1b" }
    OllamaVersion = "0.8.0"
  }
}
```

```pkl
// resources/llm.pkl
ActionID = "llmResource"
Name = "Structured Response Generator"
Description = "Generates structured JSON output."
Category = "analytics"
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/structured" }
  AllowedParams { "text" }
  PreflightCheck {
    Validations { "@(request.data().text)" != "" }
    Retry = false
    RetryTimes = 1
  }
  Chat {
    model = "llama3.2:1b"
    role = "assistant"
    prompt = "Analyze this text and return a structured response: @(request.data().text)"
    JSONResponse = true
    JSONResponseKeys { "summary"; "keywords" }
    TimeoutDuration = 60.s
  }
}
```
</details>

<details>
  <summary>🔄 Items iteration</summary>
  Iterate over multiple items in a resource to process them sequentially, using <a href="https://kdeps.com/getting-started/resources/items.html">items iteration</a> with `item.current()`, `item.prev()`, and `item.next()`.

```pkl
// workflow.pkl
AgentID = "mtvScenarioGenerator"
Description = "Generates MTV video scenarios based on song lyrics."
Version = "1.0.0"
TargetActionID = "responseResource"
Settings {
  RateLimitMax = 100
  Environment = "production"
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/mtv-scenarios"; Method = "GET" }
    }
    cors { enableCORS = true; allowOrigins { "http://localhost:8080" } }
  }
  AgentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2:1b" }
    OllamaVersion = "0.8.0"
  }
}
```

```pkl
// resources/llm.pkl
ActionID = "llmResource"
Name = "MTV Scenario Generator"
Description = "Generates MTV video scenarios for song lyrics."
Category = "ai"
Items {
  "A long, long time ago"
  "I can still remember"
  "How that music used to make me smile"
  "And I knew if I had my chance"
}
Run {
  RestrictToHTTPMethods { "GET" }
  RestrictToRoutes { "/api/v1/mtv-scenarios" }
  SkipCondition {
    "@(item.current())" == "And I knew if I had my chance" // Skip this lyric
  }
  Chat {
    model = "llama3.2:1b"
    role = "assistant"
    prompt = """
    Based on the lyric @(item.current()) from the song "American Pie," generate a suitable scenario for an MTV music video. The scenario should include a vivid setting, key visual elements, and a mood that matches the lyric's tone.
    """
    scenario {
      new { role = "system"; prompt = "You are a creative director specializing in music video production." }
    }
    JSONResponse = true
    JSONResponseKeys { "setting"; "visual_elements"; "mood" }
    TimeoutDuration = 60.s
  }
}
```

```pkl
// resources/response.pkl
ActionID = "responseResource"
Name = "API Response"
Description = "Returns MTV video scenarios."
Category = "output"
Requires { "llmResource" }
Run {
  RestrictToHTTPMethods { "GET" }
  RestrictToRoutes { "/api/v1/mtv-scenarios" }
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
AgentID = "docAnalysisAgent"
Description = "Analyzes uploaded documents with LLM."
Version = "1.0.0"
TargetActionID = "responseResource"
Settings {
  RateLimitMax = 100
  Environment = "production"
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/doc-analyze"; Method = "POST" }
    }
  }
  AgentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2-vision" }
    OllamaVersion = "0.8.0"
  }
}
```

```pkl
// resources/llm.pkl
ActionID = "llmResource"
Name = "Document Analyzer"
Description = "Extracts text from uploaded documents."
Category = "ai"
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/doc-analyze" }
  PreflightCheck {
    Validations { "@(request.filecount())" > 0 }
    Retry = false
    RetryTimes = 1
  }
  Chat {
    model = "llama3.2-vision"
    role = "assistant"
    prompt = "Extract key information from this document."
    files { "@(request.files()[0])" }
    JSONResponse = true
    JSONResponseKeys { "key_info" }
    TimeoutDuration = 60.s
  }
}
```
</details>

<details>
  <summary>🔄 Reusable AI agents</summary>
  Create flexible workflows with <a href="https://kdeps.com/getting-started/resources/remix.html">reusable AI agents</a>.

```pkl
// workflow.pkl
AgentID = "docAnalysisAgent"
Description = "Analyzes uploaded documents with LLM."
Version = "1.0.0"
TargetActionID = "responseResource"
Workflows { "@ticketResolutionAgent" }
Settings {
  RateLimitMax = 100
  Environment = "production"
  APIServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/doc-analyze"; Method = "POST" }
    }
  }
  AgentSettings {
    timezone = "Etc/UTC"
    models { "llama3.2-vision" }
    OllamaVersion = "0.8.0"
  }
}
```

```pkl
// resources/response.pkl
ActionID = "responseResource"
Name = "API Response"
Description = "Returns defect analysis result."
Category = "output"
Requires {
  "llmResource"
  "@ticketResolutionAgent/llmResource:1.0.0"
}
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/doc-analyze" }
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
ActionID = "pythonResource"
Name = "Data Formatter"
Description = "Formats extracted data for storage."
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/scan-document" }
  python {
    script = """
import pandas as pd

def format_data(data):
  df = pd.DataFrame([data])
  return df.to_json()

print(format_data(@(llm.response('llmResource'))))
"""
    TimeoutDuration = 60.s
  }
}
```
</details>

<details>
  <summary>🌍 Make API calls</summary>
  Perform API calls directly from configuration, as detailed in the <a href="https://kdeps.com/getting-started/resources/client.html">client documentation</a>.

```pkl
// resources/http_client.pkl
ActionID = "httpResource"
Name = "DMS Submission"
Description = "Submits extracted data to document management system."
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/scan-document" }
  HTTPClient {
    Method = "POST"
    url = "https://dms.example.com/api/documents"
    data { "@(python.stdout('pythonResource'))" }
    headers { ["Authorization"] = "Bearer @(session.getRecord('dms_token'))" }
    TimeoutDuration = 30.s
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
RestrictToHTTPMethods { "POST" }
RestrictToRoutes { "/api/v1/scan-document" }
PreflightCheck {
  Validations { "@(request.filetype('document'))" == "image/jpeg" }
}
SkipCondition { "@(request.data().query.length)" < 5 }
```
</details>

<details>
  <summary>📁 Serve static websites or reverse-proxied apps</summary>
  Host <a href="https://kdeps.com/getting-started/configuration/workflow.html#static-file-serving">static websites</a> or <a href="https://kdeps.com/getting-started/configuration/workflow.html#reverse-proxying">reverse-proxied apps</a> directly.

```pkl
// workflow.pkl
AgentID = "frontendAIApp"
Description = "Pairs an AI API with a Streamlit frontend for text summarization."
Version = "1.0.0"
TargetActionID = "responseResource"
Settings {
  RateLimitMax = 200
  Environment = "development"
  APIServerMode = true
  WebServerMode = true
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000
    routes {
      new { path = "/api/v1/summarize"; Method = "POST" }
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
  AgentSettings {
    timezone = "Etc/UTC"
    pythonPackages { "streamlit" }
    models { "llama3.2:1b" }
    OllamaVersion = "0.8.0"
  }
}
```
</details>

<details>
  <summary>💾 Manage state with memory operations</summary>
  Store, retrieve, and clear persistent data using <a href="https://kdeps.com/getting-started/resources/memory.html">memory operations</a>.

```pkl
expr {
  "@(memory.setRecord('user_data', request.data().data))"
}
local user_data = "@(memory.getRecord('user_data'))"
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
    new { path = "/api/v1/proxy"; Method = "GET" }
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
ActionID = "execResource"
Name = "Shell Script Runner"
Description = "Runs a shell script."
Run {
  exec {
    command = """
echo "Processing request at $(date)"
"""
    TimeoutDuration = 60.s
  }
}
```
</details>

<details>
  <summary>📦 Install Ubuntu packages</summary>
  Install <a href="https://kdeps.com/getting-started/configuration/workflow.html#ubuntu-packages">Ubuntu packages</a> via configuration for customized environments.

```pkl
// workflow.pkl
AgentSettings {
  timezone = "Etc/UTC"
  packages {
    "tesseract-ocr"
    "poppler-utils"
    "npm"
    "ffmpeg"
  }
  OllamaVersion = "0.8.0"
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
