# Functions & Utilities

Utility resources and functions that enhance and extend your AI agent workflows. These tools provide essential functionality for data manipulation, expression evaluation, type checking, and cross-resource operations.

## Available Functions & Utilities

### **Resource Functions (`functions.md`)**
**Utility Functions Specific to Core and Advanced Resources**

Resource functions provide specialized utilities for working with specific resource types. Features include:
- Resource-specific data extraction and manipulation
- Output formatting and transformation utilities
- Resource state management and control
- Error handling and validation helpers
- Performance optimization utilities

**Key Function Categories**:
- **LLM Functions**: `llm.response()`, `llm.tokens()`, `llm.cost()`
- **Client Functions**: `client.responseBody()`, `client.headers()`, `client.status()`
- **Python Functions**: `python.stdout()`, `python.stderr()`, `python.exitCode()`
- **Exec Functions**: `exec.stdout()`, `exec.stderr()`, `exec.exitCode()`

**Use Cases**: Data extraction, response processing, resource coordination, error handling

### **Global Functions (`global-functions.md`)**
**Cross-Resource Utility Functions and Helpers**

Global functions provide utilities that work across all resource types and contexts. Capabilities include:
- Universal data transformation utilities
- Cross-resource communication helpers
- System-level utilities and information
- Mathematical and string processing functions
- Date/time manipulation and formatting

**Key Function Categories**:
- **Request Functions**: `request.data()`, `request.headers()`, `request.method()`
- **Memory Functions**: `memory.get()`, `memory.set()`, `memory.clear()`
- **Session Functions**: `session.get()`, `session.set()`, `session.clear()`
- **Document Functions**: `document.JSONParser()`, `document.XMLParser()`
- **Utility Functions**: `utils.base64()`, `utils.hash()`, `utils.uuid()`

**Use Cases**: Request processing, data storage, session management, data conversion

### **Expr Block (`expr.md`)**
**Expression Evaluation and Computation**

The Expr block provides a space for evaluating PKL expressions and performing computations. Features include:
- Side-effect operations (memory updates, logging)
- Inline computational logic
- Conditional expression evaluation
- Variable assignments and data manipulation
- Custom scripting within configurations

**Common Operations**:
- Memory and session state updates
- Conditional logic execution
- Data transformation and computation
- Logging and debugging operations
- Custom business logic implementation

**Use Cases**: State management, conditional operations, data processing, custom logic

### **Data Types (`types.md`)**
**Type Definitions and Schema Enforcement**

The Data Types system provides robust type checking and schema validation. Capabilities include:
- Strong type definitions for data structures
- Schema validation for API inputs and outputs
- Custom type creation and validation
- Runtime type checking and enforcement
- Data transformation with type safety

**Supported Types**:
- **Primitive Types**: String, Integer, Float, Boolean
- **Collection Types**: Array, Map, Set
- **Custom Types**: User-defined structures and schemas
- **Optional Types**: Nullable and optional field support
- **Union Types**: Multiple type options and polymorphism

**Use Cases**: API validation, data integrity, schema enforcement, type safety

## Function Usage Patterns

### **Resource Data Extraction**
```apl
// Extract and use data from different resources
Response {
    Data {
        "llm_response": "@(llm.response('chatResource'))"
        "api_data": "@(client.responseBody('apiResource'))"
        "processing_result": "@(python.stdout('dataResource'))"
    }
}
```

### **Memory and State Management**
```apl
// Store and retrieve data across requests
Expr {
    "@(memory.setRecord('user_session', request.data().sessionId))"
    "@(memory.setRecord('last_request', utils.timestamp()))"
}
```

### **Conditional Logic with Expression Blocks**
```apl
// Implement custom business logic
Expr {
    "if @(request.data().priority == 'high') { 
        @(memory.setRecord('priority_flag', 'urgent'))
    }"
}
```

### **Type-Safe API Responses**
```apl
// Enforce response structure with types
APIResponse {
    Success = true
    Response {
        Data {
            "user_id": "@(request.data().id)"  // String type
            "score": "@(python.stdout('scoreResource'))"  // Integer type
            "metadata": "@(memory.get('session_data'))"  // Object type
        }
    }
}
```

## Best Practices

### **Function Composition**
- Chain functions logically for complex operations
- Use promises (`@()`) correctly for async operations
- Handle errors gracefully with try-catch patterns

### **Type Safety**
- Define clear schemas for all data structures
- Use type validation for API inputs and outputs
- Implement runtime type checking for critical data

### **Performance Optimization**
- Cache expensive function results when possible
- Use memory and session storage efficiently
- Optimize expression evaluation in loops

### **Error Handling**
- Implement comprehensive error checking
- Use validation functions for input sanitization
- Provide meaningful error messages and codes

## Integration Examples

### **Full-Stack AI Application**
```apl
// Combines multiple function types for complete workflow
ActionID = "fullStackResource"
Requires { "llmResource"; "validationResource" }
Run {
    // Type validation
    Validation {
        Schema = userInputSchema
        Data = "@(request.data())"
    }
    
    // State management
    Expr {
        "@(session.setRecord('current_step', 'processing'))"
    }
    
    // API response with resource functions
    APIResponse {
        Success = true
        Response {
            Data {
                "processed": "@(llm.response('llmResource'))"
                "session": "@(session.get('current_step'))"
                "timestamp": "@(utils.timestamp())"
            }
        }
    }
}
```

## Next Steps

- **[Core Resources](../core-resources/README.md)**: Learn about the basic building blocks
- **[Workflow Control](../workflow-control/README.md)**: Add conditional logic and flow control
- **[Data & Memory](../data-memory/README.md)**: Explore data storage and management options

See each documentation file for detailed function references, examples, and implementation guides. 