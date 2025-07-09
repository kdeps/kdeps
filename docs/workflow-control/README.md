# Workflow Control

Resources and features that manage the flow, logic, and validation of your AI agent workflows. These tools provide sophisticated control mechanisms to create complex, conditional, and reliable AI applications.

## Available Workflow Control Resources

### **Graph Dependencies (`kartographer.md`)**
**Resource Dependency Management and Execution Order**

The Graph Dependency system (Kartographer) manages resource execution order through dependency relationships. Features include:
- Automatic dependency resolution and topological sorting
- Parallel execution of independent resources
- Circular dependency detection and prevention
- Resource lifecycle management
- Error propagation through dependency chains

**Key Capabilities**:
- Define complex dependency graphs with multiple levels
- Conditional dependencies based on runtime conditions
- Resource grouping and batch execution
- Error handling and fallback resource chains

**Use Cases**: Complex workflows, resource orchestration, pipeline management, error handling chains

**Example Patterns**:
```apl
// Sequential processing chain
Requires { "validationResource"; "processingResource"; "outputResource" }

// Parallel processing with convergence
Requires { "dataSourceA"; "dataSourceB" } // → "aggregatorResource"
```

### **Skip Conditions (`skip.md`)**
**Conditional Execution Logic and Flow Control**

Skip Conditions provide powerful logic to conditionally execute or skip resources based on runtime data. Capabilities include:
- Dynamic condition evaluation using PKL expressions
- Request data, headers, and context-based conditions
- Resource dependency and state-based logic
- Multiple condition combinations (AND, OR, NOT)
- Custom condition functions and validators

**Common Condition Types**:
- **Route-based**: Skip based on API route or HTTP method
- **Data-based**: Skip based on request payload content
- **State-based**: Skip based on memory or session state
- **Time-based**: Skip based on time, date, or schedule
- **Permission-based**: Skip based on authentication or authorization

**Use Cases**: Conditional workflows, feature flags, A/B testing, permission-based access, environment-specific logic

**Example Applications**:
```apl
// Route-specific execution
SkipCondition { request.path() != "/api/v1/premium" }

// Feature flag implementation
SkipCondition { memory.get("feature_enabled") != "true" }

// Environment-based execution
SkipCondition { environment.get("NODE_ENV") == "development" }
```

### **Promise Operations (`promise.md`)**
**Asynchronous and Parallel Processing**

Promise Operations enable asynchronous execution and advanced resource coordination. Features include:
- Asynchronous resource execution with `@()` syntax
- Promise chaining and composition
- Parallel promise execution and synchronization
- Error handling and promise rejection
- Timeout management and cancellation

**Promise Patterns**:
- **Simple Promises**: `"@(resource.function('id'))"`
- **Promise Chains**: Sequential promise execution
- **Promise.all**: Parallel execution with synchronization
- **Promise Racing**: First-complete resource selection
- **Error Boundaries**: Promise error isolation and handling

**Use Cases**: Parallel data fetching, async operations, resource coordination, performance optimization

### **Preflight Validations (`validations.md`)**
**Input Validation and Data Integrity**

Preflight Validations ensure data quality and integrity before resource execution. Capabilities include:
- Schema-based validation using type definitions
- Custom validation rules and business logic
- Multi-level validation (request, resource, output)
- Validation error handling and reporting
- Data sanitization and transformation

**Validation Types**:
- **Schema Validation**: Type checking, required fields, format validation
- **Business Rules**: Custom logic validation, constraint checking
- **Security Validation**: Input sanitization, injection prevention
- **Data Integrity**: Consistency checks, referential integrity
- **Performance Validation**: Size limits, complexity bounds

**Use Cases**: API security, data quality assurance, business rule enforcement, input sanitization

### **API Request Validations (`api-request-validations.md`)**
**Request Filtering and Security**

API Request Validations provide security and filtering for incoming requests. Features include:
- Request authentication and authorization
- Rate limiting and throttling
- Input filtering and sanitization
- Request size and complexity limits
- IP-based filtering and geo-blocking

**Security Features**:
- **Authentication**: Token validation, API key verification
- **Authorization**: Role-based access control, permission checking
- **Rate Limiting**: Request frequency and volume control
- **Input Filtering**: SQL injection, XSS, and payload validation
- **Monitoring**: Request logging, anomaly detection, audit trails

**Use Cases**: API security, abuse prevention, access control, compliance, monitoring

## Workflow Control Patterns

### **Linear Pipeline**
```apl
// Simple sequential processing
ActionID = "finalResource"
Requires { "inputResource" }
// inputResource → validationResource → processingResource → finalResource
```

### **Conditional Branching**
```apl
// Different paths based on conditions
ActionID = "routerResource"
SkipCondition { request.data().type != "premium" }
Requires { "premiumProcessingResource" }
// Alternative: "standardProcessingResource"
```

### **Parallel Processing with Convergence**
```apl
// Multiple parallel resources feeding into final aggregation
ActionID = "aggregatorResource"
Requires { "dataSourceA"; "dataSourceB"; "dataSourceC" }
// All sources process in parallel, then aggregate results
```

### **Error Handling Chain**
```apl
// Primary resource with fallback options
ActionID = "primaryResource"
// If primaryResource fails → fallbackResource → errorHandlerResource
Requires { "fallbackResource" }
```

### **Validation Pipeline**
```apl
// Multi-stage validation and processing
ActionID = "outputResource"
Requires { "authValidation"; "inputValidation"; "businessValidation"; "processingResource" }
```

## Advanced Control Patterns

### **Feature Flag System**
```apl
// Implement feature toggles across workflows
SkipCondition { 
    memory.get("features").newFeatureEnabled != true 
}
```

### **A/B Testing Framework**
```apl
// Route users to different resource variants
SkipCondition { 
    utils.hash(request.data().userId) % 2 == 0 
}
```

### **Circuit Breaker Pattern**
```apl
// Skip expensive resources when error rate is high
SkipCondition { 
    memory.get("error_rate") > 0.1 
}
```

### **Environment-Specific Logic**
```apl
// Different behavior in different environments
SkipCondition { 
    environment.get("DEPLOYMENT_ENV") != "production" 
}
```

## Performance Optimization

### **Resource Parallelization**
- Identify independent resources for parallel execution
- Use dependency graphs to maximize parallelism
- Balance resource usage with execution speed

### **Conditional Execution**
- Use skip conditions to avoid unnecessary processing
- Implement early exit patterns for efficiency
- Cache validation results when possible

### **Error Handling Strategy**
- Design fallback chains for critical workflows
- Implement circuit breakers for external dependencies
- Use validation to prevent expensive error conditions

## Best Practices

### **Dependency Design**
- Keep dependency graphs simple and understandable
- Avoid deep nesting and complex interdependencies
- Use clear naming conventions for resources

### **Validation Strategy**
- Validate early and often in the workflow
- Use schema validation for structure, custom validation for business rules
- Provide clear, actionable error messages

### **Performance Considerations**
- Design for parallel execution when possible
- Use skip conditions to optimize resource usage
- Implement appropriate timeout and retry policies

### **Security Implementation**
- Validate all inputs at API boundaries
- Implement proper authentication and authorization
- Use rate limiting to prevent abuse

## Integration Examples

### **Secure API Workflow**
```apl
// Complete secure processing workflow
ActionID = "secureApiResource"
Requires { "authValidation"; "rateLimitCheck"; "inputValidation"; "businessLogic" }

// Authentication check
SkipCondition { request.headers().Authorization == null }

// Rate limiting
SkipCondition { memory.get("request_count") > 100 }

// Input validation with schema
Validation {
    Schema = apiInputSchema
    Data = "@(request.data())"
}
```

## Next Steps

- **[Core Resources](../core-resources/README.md)**: Learn about basic building blocks to control
- **[Functions & Utilities](../functions-utilities/README.md)**: Explore helper functions for workflow logic
- **[Data & Memory](../data-memory/README.md)**: Understand state management for workflow control

See each documentation file for detailed configuration options, examples, and implementation patterns. 