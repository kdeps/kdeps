---
outline: deep
---

# Expr Block

The `Expr` block provides a dedicated space for evaluating standard PKL expressions within Kdeps resources. It is primarily designed for executing expressions that produce side effects, such as updating resources or triggering actions, while also supporting general-purpose evaluation of any valid PKL expression for inline logic and scripting within configurations.

## Overview of the `Expr` Block

The `Expr` block is designed to evaluate PKL expressions in a straightforward and flexible manner. Its key uses include:

- **Side-Effect Operations**: Executing functions like `memory.setRecord()` that modify resources or state without returning significant values for further processing.

- **Inline Scripting**: Evaluating arbitrary PKL expressions to implement logic, assignments, or procedural tasks directly within a configuration.

- **State Management**: Updating memory, session, or temporary variables during resource execution.

- **Conditional Logic**: Implementing business rules and conditional operations based on runtime data.

The `Expr` block simplifies the execution of side-effect operations that don't need their results to be directly used in response data.

## Syntax and Usage

The `Expr` block is defined as follows:

```apl
Expr {
  // Valid PKL expression(s)
  // Multiple expressions are evaluated in sequence
}
```

Each expression within the block is evaluated in sequence, allowing multiple expressions to form a procedural sequence when needed. The expressions can access all available functions, variables, and data within the resource context.

### Basic Example

The `Expr` block is well-suited for operations that update state, such as setting memory items:

```apl
ActionID = "exprResource"
Name = "Expression Evaluator"
Description = "Evaluate PKL expressions for state management"
Category = "utility"
Run {
  Expr {
    "@(memory.setRecord("status", "active"))"
  }
}
```

In this example, the memory store is updated to indicate an active status. The `memory.setRecord()` function is executed as a side effect, and no return value is required for further processing.

## Common Use Cases

### Memory Operations

Store and manage persistent data across requests:

```apl
Expr {
  "@(memory.setRecord("user_id", "12345"))"
  "@(memory.setRecord("session_start", utils.timestamp()))"
  "@(memory.setRecord("request_count", memory.get("request_count") + 1))"
}
```

### Session Management

Handle session-specific data and state:

```apl
Expr {
  "@(session.setRecord("current_step", "processing"))"
  "@(session.setRecord("attempts", "3"))"
  "@(session.setRecord("last_activity", utils.timestamp()))"
}
```

### Conditional Logic

Implement business rules and conditional operations:

```apl
Expr {
  "if @(request.data().priority == "high") { 
    @(memory.setRecord("priority", "urgent"))
    @(memory.setRecord("escalation_needed", true))
  } else {
    @(memory.setRecord("priority", "normal"))
  }"
}
```

### Data Processing

Transform and process data before storage:

```apl
Expr {
  "local processedData = @(document.JSONParser(request.data().json))"
  "@(memory.setRecord("processed_data", processedData))"
  "@(memory.setRecord("processing_timestamp", utils.timestamp()))"
  "@(memory.setRecord("data_size", processedData.length))"
}
```

### Logging and Debugging

Track execution flow and debug information:

```apl
Expr {
  "@(logger.info("Processing request for user: " + request.data().userId))"
  "@(memory.setRecord("debug_trace", memory.get("debug_trace") + [utils.timestamp()]))"
}
```

## Advanced Patterns

### Complex Conditional Logic

Handle multiple conditions and branching logic:

```apl
Expr {
  "local userType = @(request.data().userType)"
  "local requestData = @(request.data())"
  
  "if userType == "premium" {
    @(memory.setRecord("rate_limit", 1000))
    @(memory.setRecord("features", ["advanced", "priority", "analytics"]))
  } else if userType == "standard" {
    @(memory.setRecord("rate_limit", 100))
    @(memory.setRecord("features", ["basic", "standard"]))
  } else {
    @(memory.setRecord("rate_limit", 10))
    @(memory.setRecord("features", ["basic"]))
  }"
}
```

### Data Validation and Preprocessing

Validate and prepare data before processing:

```apl
Expr {
  "local email = @(request.data().email)"
  "local isValidEmail = email.contains("@") && email.contains(".")"
  
  "if isValidEmail {
    @(memory.setRecord("user_email", email))
    @(memory.setRecord("validation_status", "passed"))
  } else {
    @(memory.setRecord("validation_status", "failed"))
    @(memory.setRecord("error_message", "Invalid email format"))
  }"
}
```

### Batch Operations

Perform multiple related operations efficiently:

```apl
Expr {
  "local userData = @(request.data())"
  "local timestamp = @(utils.timestamp())"
  
  // Batch memory operations
  "@(memory.setBatch({
    "user_profile": userData,
    "last_login": timestamp,
    "session_active": true,
    "login_count": memory.get("login_count") + 1
  }))"
}
```

### Error Handling

Implement error handling and recovery logic:

```apl
Expr {
  "try {
    local result = @(client.responseBody("externalAPI"))
    @(memory.setRecord("api_result", result))
    @(memory.setRecord("api_status", "success"))
  } catch (error) {
    @(memory.setRecord("api_status", "failed"))
    @(memory.setRecord("error_message", error.message))
    @(logger.error("API call failed: " + error.message))
  }"
}
```

## Best Practices

### Expression Organization

- **Group Related Operations**: Keep related expressions together in logical blocks
- **Use Clear Variable Names**: Make expressions self-documenting with descriptive names
- **Break Complex Logic**: Split complex expressions into multiple, simpler ones

### Performance Considerations

- **Minimize External Calls**: Avoid unnecessary API calls or expensive operations in loops
- **Cache Computed Values**: Store frequently used calculations in variables
- **Batch Operations**: Use batch functions when updating multiple memory records

### Error Handling

- **Validate Inputs**: Check data validity before processing
- **Graceful Degradation**: Handle errors without breaking the entire workflow
- **Meaningful Error Messages**: Provide clear, actionable error information

### Memory Management

- **Clean Up Temporary Data**: Remove temporary variables and data when no longer needed
- **Use Appropriate Storage**: Choose between memory, session, and temporary storage based on data lifecycle
- **Monitor Memory Usage**: Be mindful of memory consumption with large datasets

## Integration with Other Resources

### Using Expr with LLM Resources

Prepare data for LLM processing:

```apl
ActionID = "llmPreProcessor"
Requires { "dataProcessor" }
Run {
  Expr {
    "local userData = @(memory.get("user_profile"))"
    "local context = "User is a " + userData.type + " with " + userData.experience + " experience""
    "@(memory.setRecord("llm_context", context))"
  }
  
  Chat {
    Model = "llama3.2:1b"
    Prompt = "@(memory.get("llm_context")) + ": " + @(request.data().query)"
    JSONResponse = true
  }
}
```

### Using Expr with HTTP Client Resources

Process API responses:

```apl
ActionID = "apiProcessor"
Requires { "externalApiCall" }
Run {
  Expr {
    "local apiResponse = @(client.responseBody("externalApiCall"))"
    "local processedData = apiResponse.data.map(item => {
      processedItem = item + {timestamp: utils.timestamp()}
      return processedItem
    })"
    "@(memory.setRecord("processed_api_data", processedData))"
  }
}
```

### Using Expr with Validation

Implement custom validation logic:

```apl
ActionID = "customValidator"
Run {
  Expr {
    "local requestData = @(request.data())"
    "local isValid = true"
    "local errors = []"
    
    "if requestData.age < 18 {
      isValid = false
      errors = errors + ["Age must be 18 or older"]
    }"
    
    "if requestData.email.length == 0 {
      isValid = false
      errors = errors + ["Email is required"]
    }"
    
    "@(memory.setRecord("validation_result", {valid: isValid, errors: errors}))"
  }
}
```

## Common Pitfalls and Solutions

### Pitfall: Forgetting Promise Syntax

**Incorrect:**
```apl
Expr {
  memory.setRecord("key", "value")  // Missing @() wrapper
}
```

**Correct:**
```apl
Expr {
  "@(memory.setRecord("key", "value"))"  // Proper promise syntax
}
```

### Pitfall: Complex Nested Logic

**Problematic:**
```apl
Expr {
  "if condition1 { if condition2 { if condition3 { /* deeply nested */ } } }"
}
```

**Better:**
```apl
Expr {
  "local shouldProcess = condition1 && condition2 && condition3"
  "if shouldProcess {
    // Clear, simple logic
  }"
}
```

### Pitfall: Side Effects in Conditions

**Problematic:**
```apl
Expr {
  "if @(memory.setRecord("key", "value")) {  // Side effect in condition
    // logic
  }"
}
```

**Better:**
```apl
Expr {
  "@(memory.setRecord("key", "value"))"
  "if @(memory.get("key")) == "value" {
    // Clear separation of side effects and conditions
  }"
}
```

## Next Steps

- **[Global Functions](./global-functions.md)**: Learn about available functions for use in Expr blocks
- **[Data Types](./types.md)**: Understand type handling in expressions
- **[Memory Operations](../data-memory/memory.md)**: Master persistent data management
- **[Workflow Control](../workflow-control/README.md)**: Combine Expr with conditional logic and validation

The Expr block is a powerful tool for implementing custom logic and state management in your Kdeps AI agents. Use it to create sophisticated workflows that respond dynamically to data and conditions.
