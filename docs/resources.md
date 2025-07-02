---
outline: deep
---

# Resources Overview

Resources are the building blocks of Kdeps AI agents. Each resource represents a specific functionality that can be composed together to create complex workflows. All resources follow the v0.3.1 schema with capitalized property names.

## 🔧 Core Resources

These are the fundamental resources that form the backbone of most AI agents:

### **LLM Resource** (`llm.md`)
- **Purpose**: Interact with language models
- **Key Features**: Chat completion, JSON responses, tool calling
- **Common Use Cases**: Text generation, conversation, structured output
- **Schema Properties**: `ActionID`, `Chat`, `Model`, `Prompt`, `JSONResponse`

### **API Response Resource** (`response.md`)
- **Purpose**: Return structured JSON responses
- **Key Features**: Success/error handling, metadata, headers
- **Common Use Cases**: API endpoints, standardized responses
- **Schema Properties**: `ActionID`, `APIResponse`, `Success`, `Response`, `Meta`

### **HTTP Client Resource** (`client.md`)
- **Purpose**: Make HTTP requests to external APIs
- **Key Features**: GET/POST requests, headers, authentication
- **Common Use Cases**: API integration, data fetching
- **Schema Properties**: `ActionID`, `Client`, `Method`, `URL`, `Headers`

### **Python Resource** (`python.md`)
- **Purpose**: Execute Python scripts
- **Key Features**: Script execution, input/output handling
- **Common Use Cases**: Data processing, calculations, ML inference
- **Schema Properties**: `ActionID`, `Python`, `Script`, `Input`, `Output`

### **Exec Resource** (`exec.md`)
- **Purpose**: Execute shell commands
- **Key Features**: Command execution, environment variables
- **Common Use Cases**: System operations, file management
- **Schema Properties**: `ActionID`, `Exec`, `Command`, `Args`, `Env`

## 🛠️ Advanced Resources

These resources provide specialized functionality for complex use cases:

### **Multi-Modal LLM Models** (`multimodal.md`)
- **Purpose**: Process images and text together
- **Key Features**: Vision models, file uploads, image analysis
- **Common Use Cases**: Image description, visual QA, document analysis
- **Schema Properties**: `ActionID`, `Chat`, `Files`, `Model`

### **AI Image Generators** (`image-generators.md`)
- **Purpose**: Generate images using AI models
- **Key Features**: Stable Diffusion, prompt-based generation
- **Common Use Cases**: Content creation, image synthesis
- **Schema Properties**: `ActionID`, `ImageGenerator`, `Model`, `Prompt`

### **Tool Calling (MCP)** (`tools.md`)
- **Purpose**: Integrate external tools and functions
- **Key Features**: Open-source tool calling, function definitions
- **Common Use Cases**: Database queries, file operations, API calls
- **Schema Properties**: `ActionID`, `Tools`, `Name`, `Script`, `Parameters`

### **Items Iteration** (`items.md`)
- **Purpose**: Process multiple items in batches
- **Key Features**: Batch processing, iteration control
- **Common Use Cases**: Data processing, bulk operations
- **Schema Properties**: `ActionID`, `Items`, `Item`, `Collection`

## 🔗 Workflow Control

These resources help manage the flow and logic of your AI agents:

### **Graph Dependency** (`kartographer.md`)
- **Purpose**: Define resource dependencies and execution order
- **Key Features**: Dependency graphs, parallel execution
- **Common Use Cases**: Complex workflows, resource chaining
- **Schema Properties**: `ActionID`, `Kartographer`, `Dependencies`

### **Skip Conditions** (`skip.md`)
- **Purpose**: Conditionally skip resource execution
- **Key Features**: Conditional logic, early termination
- **Common Use Cases**: Error handling, optimization
- **Schema Properties**: `ActionID`, `Skip`, `Condition`

### **Preflight Validations** (`validations.md`)
- **Purpose**: Validate inputs before processing
- **Key Features**: Input validation, error reporting
- **Common Use Cases**: Data validation, security checks
- **Schema Properties**: `ActionID`, `Validations`, `Rules`

### **API Request Validations** (`api-request-validations.md`)
- **Purpose**: Validate API requests
- **Key Features**: Request filtering, method restrictions
- **Common Use Cases**: API security, input sanitization
- **Schema Properties**: `ActionID`, `RestrictToHTTPMethods`, `RestrictToRoutes`

### **Promise Operator** (`promise.md`)
- **Purpose**: Handle asynchronous operations
- **Key Features**: Promise resolution, error handling
- **Common Use Cases**: Async workflows, parallel processing
- **Schema Properties**: `ActionID`, `Promise`, `Resolve`, `Reject`

## 💾 Data & Memory

These resources handle data persistence and file operations:

### **Memory Operations** (`memory.md`)
- **Purpose**: Persistent data storage and retrieval
- **Key Features**: Key-value storage, session management
- **Common Use Cases**: User sessions, caching, state management
- **Schema Properties**: `ActionID`, `Memory`, `Key`, `Value`

### **Data Folder** (`data.md`)
- **Purpose**: File and folder management
- **Key Features**: File operations, directory management
- **Common Use Cases**: File processing, data storage
- **Schema Properties**: `ActionID`, `Data`, `Path`, `Operation`

### **Working with JSON** (`json.md`)
- **Purpose**: JSON processing and manipulation
- **Key Features**: Parsing, generation, transformation
- **Common Use Cases**: Data transformation, API responses
- **Schema Properties**: `ActionID`, `JSON`, `Parse`, `Generate`

### **File Uploads** (`files.md`)
- **Purpose**: Handle file uploads and processing
- **Key Features**: File upload, validation, processing
- **Common Use Cases**: Document processing, image uploads
- **Schema Properties**: `ActionID`, `Files`, `Upload`, `Process`

## ⚡ Functions & Utilities

These provide utility functions and expressions:

### **Resource Functions** (`functions.md`)
- **Purpose**: Resource-specific utility functions
- **Key Features**: Resource access, data manipulation
- **Common Use Cases**: Cross-resource communication, data access
- **Schema Properties**: Various function-specific properties

### **Global Functions** (`global-functions.md`)
- **Purpose**: Cross-resource utility functions
- **Key Features**: Global utilities, helper functions
- **Common Use Cases**: Common operations, utilities
- **Schema Properties**: Various function-specific properties

### **Expr Block** (`expr.md`)
- **Purpose**: Expression evaluation and computation
- **Key Features**: Mathematical operations, logic evaluation
- **Common Use Cases**: Calculations, conditional logic
- **Schema Properties**: `ActionID`, `Expr`, `Expression`

### **Data Types** (`types.md`)
- **Purpose**: Define and validate data types
- **Key Features**: Type definitions, validation
- **Common Use Cases**: Schema validation, type safety
- **Schema Properties**: `ActionID`, `Types`, `Definition`

## 🔄 Reusability

### **Reusing and Remixing AI Agents** (`remix.md`)
- **Purpose**: Compose and reuse AI agent components
- **Key Features**: Agent composition, modular design
- **Common Use Cases**: Building complex agents, code reuse
- **Schema Properties**: `ActionID`, `Remix`, `Components`

## 📋 Resource Schema Structure

All resources follow a consistent schema structure in v0.3.1:

```apl
ActionID = "uniqueResourceID"
Name = "Human Readable Name"
Description = "Resource description"
Category = "resource_category"
Requires { "dependency1"; "dependency2" }  // Optional dependencies

Run {
  // Resource-specific configuration
  RestrictToHTTPMethods { "GET"; "POST" }  // Optional HTTP method restrictions
  RestrictToRoutes { "/api/v1/endpoint" }  // Optional route restrictions
  
  // Resource-specific blocks (e.g., Chat, Client, Python, etc.)
  ResourceSpecificBlock {
    // Configuration properties
  }
}
```

## 🎯 Choosing the Right Resources

### **For Simple APIs**
- **LLM Resource** + **API Response Resource**

### **For Data Processing**
- **Python Resource** + **HTTP Client Resource** + **API Response Resource**

### **For File Operations**
- **Data Folder** + **Python Resource** + **API Response Resource**

### **For Complex Workflows**
- **Graph Dependency** + **Multiple Resources** + **Skip Conditions**

### **For Multi-Modal Applications**
- **Multi-Modal LLM** + **Image Generators** + **API Response Resource**

## 📚 Next Steps

1. **Start with Core Resources**: Begin with LLM and API Response resources
2. **Add Advanced Features**: Incorporate tools, validations, and workflow control
3. **Optimize with Utilities**: Use functions and expressions for efficiency
4. **Scale with Composition**: Leverage reusability features for complex applications

Explore the individual resource documentation for detailed examples and configuration options.
