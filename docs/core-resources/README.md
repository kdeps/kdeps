# Core Resources

These are the fundamental building blocks for Kdeps AI agents. Each resource provides essential capabilities that can be composed into sophisticated workflows and applications.

## Available Core Resources

### **LLM Resource (`llm.md`)**
**Language Model Interactions and Chat Functionality**

The LLM resource enables interaction with large language models through Ollama. It supports:
- Text generation and conversation
- Structured JSON output with schema enforcement
- Multi-turn chat sessions with context
- Custom prompts and role definitions
- Timeout management and error handling

**Use Cases**: Chatbots, content generation, data analysis, question answering, code generation

### **API Response Resource (`response.md`)**
**Structured JSON Responses for API Endpoints**

The Response resource formats and returns structured JSON responses for API endpoints. Features include:
- Standardized success/error response format
- Flexible data composition from multiple resources
- Error handling and status code management
- Header customization and metadata support

**Use Cases**: API development, web services, mobile app backends, webhook responses

### **HTTP Client Resource (`client.md`)**
**External API Calls and Web Service Integration**

The Client resource enables HTTP requests to external APIs and services. Capabilities include:
- GET, POST, PUT, DELETE, and other HTTP methods
- Authentication header management
- Request/response body handling
- Error handling and retry logic
- JSON and form data support

**Use Cases**: Third-party API integration, microservices communication, data fetching, webhook triggers

### **Python Resource (`python.md`)**
**Python Script Execution for Data Processing**

The Python resource executes Python scripts within the Kdeps environment. Features include:
- Custom script execution with full Python capabilities
- Package management with pip and conda
- Environment variable access
- File system operations
- Data processing and machine learning workflows

**Use Cases**: Data analysis, machine learning inference, file processing, algorithmic computations, database operations

### **Exec Resource (`exec.md`)**
**Shell Command Execution and System Operations**

The Exec resource runs shell commands for system-level operations. Provides:
- Command execution with output capture
- Environment variable management
- Working directory control
- Error handling and exit code checking
- Process timeout management

**Use Cases**: File operations, system administration, git operations, deployment scripts, utility functions

## Getting Started

1. **Choose the Right Resource**: Start with the resource that matches your primary use case
2. **Review Examples**: Each resource documentation includes practical configuration examples
3. **Understand Dependencies**: Resources can depend on each other to create complex workflows
4. **Test Incrementally**: Build and test resources individually before combining them

## Resource Composition

Core resources can be combined using the dependency system:

```apl
// Example: API that uses LLM and returns structured response
Requires { "llmResource"; "pythonResource" }
```

This creates execution order: `pythonResource` → `llmResource` → `responseResource`

## Next Steps

- **[Functions & Utilities](../functions-utilities/README.md)**: Learn about helper functions and utilities
- **[Workflow Control](../workflow-control/README.md)**: Add conditional logic and advanced flow control
- **[Advanced Resources](../advanced-resources/README.md)**: Explore specialized capabilities like image generation and multimodal models

See each resource's documentation file for detailed configuration options, examples, and best practices. 