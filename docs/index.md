---
outline: deep
---

# Kdeps v0.3.1 - AI Agent Framework

**A robust framework for building AI agents with enhanced schema structure and improved developer experience.**

## 🚀 What's New in v0.3.1

- **Enhanced Schema Structure**: All PKL properties now use capitalized naming (e.g., `AgentID`, `Settings`, `ActionID`)
- **Improved Rate Limiting**: New `RateLimitMax` property for API throttling control
- **Better Resource Organization**: Streamlined resource types and improved workflow control
- **Enhanced Documentation**: Comprehensive guides with practical examples

## 🎯 Quick Overview

Kdeps enables you to build AI agents that can:

- **Process API requests** with structured JSON responses
- **Interact with LLMs** using open-source models like LLaMA
- **Execute Python scripts** and shell commands
- **Make HTTP requests** to external APIs
- **Handle file uploads** and process data
- **Chain multiple resources** in dependency graphs
- **Deploy as Docker containers** with full-stack capabilities

## 📖 Documentation Structure

### 🚀 Getting Started
- **[Installation](./getting-started/introduction/installation.md)** - Set up Kdeps on your system
- **[Quickstart Guide](./getting-started/introduction/quickstart.md)** - Build your first AI agent

### ⚙️ Configuration
- **[System Configuration](./getting-started/configuration/configuration.md)** - Global settings
- **[Workflow Configuration](./getting-started/configuration/workflow.md)** - Agent setup and deployment
- **[CORS Configuration](./getting-started/configuration/cors.md)** - Cross-origin request handling
- **[Web Server Configuration](./getting-started/configuration/webserver.md)** - Frontend integration

### 🔧 Core Resources
- **[LLM Resource](./getting-started/resources/llm.md)** - Language model interactions
- **[API Response Resource](./getting-started/resources/response.md)** - Structured JSON responses
- **[HTTP Client Resource](./getting-started/resources/client.md)** - External API calls
- **[Python Resource](./getting-started/resources/python.md)** - Script execution
- **[Exec Resource](./getting-started/resources/exec.md)** - Shell command execution

### 🛠️ Advanced Resources
- **[Multi-Modal LLM Models](./getting-started/resources/multimodal.md)** - Vision and text processing
- **[AI Image Generators](./getting-started/resources/image-generators.md)** - Image generation with Stable Diffusion
- **[Tool Calling (MCP)](./getting-started/resources/tools.md)** - Open-source tool integration
- **[Items Iteration](./getting-started/resources/items.md)** - Batch processing capabilities

### 🔗 Workflow Control
- **[Graph Dependency](./getting-started/resources/kartographer.md)** - Resource dependency management
- **[Skip Conditions](./getting-started/resources/skip.md)** - Conditional execution
- **[Preflight Validations](./getting-started/resources/validations.md)** - Input validation
- **[API Request Validations](./getting-started/resources/api-request-validations.md)** - Request filtering

### 💾 Data & Memory
- **[Memory Operations](./getting-started/resources/memory.md)** - Persistent data storage
- **[Data Folder](./getting-started/resources/data.md)** - File management
- **[Working with JSON](./getting-started/resources/json.md)** - JSON processing utilities
- **[File Uploads](./getting-started/tutorials/files.md)** - File handling

### ⚡ Functions & Utilities
- **[Resource Functions](./getting-started/resources/functions.md)** - Resource-specific utilities
- **[Global Functions](./getting-started/resources/global-functions.md)** - Cross-resource utilities
- **[Expr Block](./getting-started/resources/expr.md)** - Expression evaluation
- **[Data Types](./getting-started/resources/types.md)** - Supported data types

### 🔄 Reusability
- **[Reusing and Remixing AI Agents](./getting-started/resources/remix.md)** - Agent composition

### 📚 Tutorials
- **[Weather API Tutorial](./getting-started/tutorials/how-to-weather-api.md)** - Complete API example
- **[Structured LLM Responses](./getting-started/tutorials/how-to-structure-llm.md)** - JSON output formatting

## 🏗️ Architecture Overview

Kdeps follows a resource-based architecture where each component is a self-contained unit:

```apl
// workflow.pkl - Main configuration
AgentID = "myAIAgent"
Description = "A sample AI agent"
Version = "1.0.0"
TargetActionID = "responseResource"
Settings {
  RateLimitMax = 100
  Environment = "production"
  APIServerMode = true
  APIServer {
    HostIP = "127.0.0.1"
    PortNum = 3000
    Routes {
      new { Path = "/api/v1/query"; Method = "POST" }
    }
  }
  AgentSettings {
    Models { "llama3.2:1b" }
    OllamaVersion = "0.8.0"
  }
}
```

```apl
// resources/llm.pkl - LLM interaction
ActionID = "llmResource"
Name = "Language Model"
Description = "Processes queries with LLM"
Category = "ai"
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/query" }
  Chat {
    Model = "llama3.2:1b"
    Role = "assistant"
    Prompt = "Answer this question: @(request.data().query)"
    JSONResponse = true
    JSONResponseKeys { "answer"; "confidence" }
    TimeoutDuration = 60.s
  }
}
```

```apl
// resources/response.pkl - API response
ActionID = "responseResource"
Name = "API Response"
Description = "Returns structured JSON response"
Category = "output"
Requires { "llmResource" }
Run {
  RestrictToHTTPMethods { "POST" }
  RestrictToRoutes { "/api/v1/query" }
  APIResponse {
    Success = true
    Response {
      Data { "@(llm.response('llmResource'))" }
    }
    Meta { Headers { ["Content-Type"] = "application/json" } }
  }
}
```

## 🚀 Quick Start Example

1. **Install Kdeps:**
   ```bash
   curl -fsSL https://kdeps.com/install.sh | sh
   ```

2. **Create a new project:**
   ```bash
   kdeps new my-agent
   cd my-agent
   ```

3. **Configure the workflow** in `workflow.pkl`

4. **Define resources** in the `resources/` directory

5. **Package and run:**
   ```bash
   kdeps package my-agent
   kdeps run my-agent-1.0.0.kdeps
   ```

## 🔧 Key Features

### **Schema v0.3.1 Compliance**
All resources use the new capitalized property naming convention for better consistency and clarity.

### **Docker Integration**
Seamless containerization with automatic dependency management and model downloading.

### **API-First Design**
Built-in API server with CORS support, request validation, and structured JSON responses.

### **Resource Composition**
Chain multiple resources together using dependency graphs for complex workflows.

### **Open Source Focus**
Designed specifically for open-source LLMs with tool calling capabilities.

### **Developer Experience**
Comprehensive documentation, examples, and utilities for rapid development.

## 🎯 Use Cases

- **API Development**: Create AI-powered APIs with structured responses
- **Data Processing**: Build pipelines for data transformation and analysis
- **Content Generation**: Generate text, images, and structured content
- **Integration**: Connect multiple services and APIs
- **Automation**: Automate complex workflows with AI assistance

## 📚 Next Steps

1. **[Install Kdeps](./getting-started/introduction/installation.md)** to get started
2. **[Follow the Quickstart](./getting-started/introduction/quickstart.md)** to build your first agent
3. **[Explore Resources](./getting-started/resources/resources.md)** to understand the building blocks
4. **[Check out Tutorials](./getting-started/tutorials/how-to-weather-api.md)** for practical examples

---

**Ready to build your first AI agent?** Start with the [Quickstart Guide](./getting-started/introduction/quickstart.md)!
