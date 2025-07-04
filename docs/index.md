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
- **[System Configuration](./getting-started/configuration/configuration.md)** - Global settings and environment setup
- **[Workflow Configuration](./getting-started/configuration/workflow.md)** - Agent setup and deployment configuration
- **[CORS Configuration](./getting-started/configuration/cors.md)** - Cross-origin request handling
- **[Web Server Configuration](./getting-started/configuration/webserver.md)** - Frontend integration and hosting

### 🔧 Core Resources
- **[LLM Resource](./core-resources/llm.md)** - Language model interactions and chat functionality
- **[API Response Resource](./core-resources/response.md)** - Structured JSON responses for API endpoints
- **[HTTP Client Resource](./core-resources/client.md)** - External API calls and web service integration
- **[Python Resource](./core-resources/python.md)** - Python script execution and data processing
- **[Exec Resource](./core-resources/exec.md)** - Shell command execution and system operations

### 🛠️ Advanced Resources
- **[Multi-Modal LLM Models](./advanced-resources/multimodal.md)** - Vision and text processing capabilities
- **[AI Image Generators](./advanced-resources/image-generators.md)** - Image generation with Stable Diffusion
- **[Tool Calling (MCP)](./advanced-resources/tools.md)** - Open-source tool integration and function calling
- **[Items Iteration](./advanced-resources/items.md)** - Batch processing and collection handling

### 🔗 Workflow Control
- **[Graph Dependencies](./workflow-control/kartographer.md)** - Resource dependency management and execution order
- **[Skip Conditions](./workflow-control/skip.md)** - Conditional execution logic
- **[Promise Operations](./workflow-control/promise.md)** - Asynchronous and parallel processing
- **[Preflight Validations](./workflow-control/validations.md)** - Input validation and data integrity
- **[API Request Validations](./workflow-control/api-request-validations.md)** - Request filtering and security

### 💾 Data & Memory
- **[Memory Operations](./data-memory/memory.md)** - Persistent data storage and state management
- **[Data Folder](./data-memory/data.md)** - File management and project data organization
- **[Working with JSON](./data-memory/json.md)** - JSON processing utilities and data manipulation
- **[File Uploads](./data-memory/files.md)** - File handling and upload processing

### ⚡ Functions & Utilities
- **[Resource Functions](./functions-utilities/functions.md)** - Resource-specific utility functions
- **[Global Functions](./functions-utilities/global-functions.md)** - Cross-resource utility functions and helpers
- **[Expr Block](./functions-utilities/expr.md)** - Expression evaluation and computation
- **[Data Types](./functions-utilities/types.md)** - Type definitions and schema enforcement

### 🔄 Reusability
- **[Reusing and Remixing AI Agents](./reusability/remix.md)** - Agent composition and workflow sharing

### 📚 Tutorials
- **[Weather API Tutorial](./tutorials/how-to-weather-api.md)** - Complete API development example
- **[Structured LLM Responses](./tutorials/how-to-structure-llm.md)** - JSON output formatting and schema design

## 🏗️ Architecture Overview

Kdeps follows a resource-based architecture where each component is a self-contained, reusable unit that can be composed into complex workflows:

```apl
// workflow.pkl - Main configuration
AgentID = "myAIAgent"
Description = "A sample AI agent for data processing"
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

### **Docker Integration**
Seamless containerization with automatic dependency management, model downloading, and environment isolation.

### **API-First Design**
Built-in API server with CORS support, request validation, rate limiting, and structured JSON responses.

### **Resource Composition**
Chain multiple resources together using dependency graphs to create complex, reusable workflows.

### **Open Source Focus**
Designed specifically for open-source LLMs with comprehensive tool calling capabilities and model flexibility.

### **Developer Experience**
Comprehensive documentation, practical examples, and intuitive configuration for rapid development.

## 🎯 Use Cases

- **API Development**: Create AI-powered APIs with structured responses and validation
- **Data Processing**: Build pipelines for data transformation, analysis, and ETL operations
- **Content Generation**: Generate text, images, and structured content using AI models
- **Service Integration**: Connect multiple services and APIs in complex workflows
- **Process Automation**: Automate complex workflows with AI assistance and decision-making

## 📚 Next Steps

1. **[Install Kdeps](./getting-started/introduction/installation.md)** to set up your development environment
2. **[Follow the Quickstart](./getting-started/introduction/quickstart.md)** to build your first functional AI agent
3. **[Explore the Tutorials](./tutorials/README.md)** for practical, real-world examples

---

For questions, support, or contributions, visit our [GitHub repository](https://github.com/kdeps/kdeps) or check out the [community resources](./resources.md).
