# Reusability

Learn how to create reusable, composable AI agents and share workflows across projects. This section covers agent composition, workflow patterns, and best practices for building modular, maintainable AI systems.

## Overview

Kdeps promotes reusability through several key mechanisms:
- **Agent Composition**: Combine multiple agents into sophisticated workflows
- **Resource Sharing**: Reuse common resources across different projects
- **Workflow Templates**: Create reusable workflow patterns and configurations
- **Package Distribution**: Share and distribute agents as `.kdeps` packages

## Agent Composition Patterns

### **[Reusing and Remixing AI Agents (`remix.md`)](./remix.md)**
**Agent Composition and Workflow Sharing**

The remix system enables sophisticated agent composition and workflow sharing. Key capabilities include:

- **Agent Integration**: Combine multiple packaged agents into unified workflows
- **Workflow Inheritance**: Extend existing agents with additional capabilities
- **Resource Composition**: Mix and match resources from different agents
- **Configuration Overrides**: Customize inherited agents for specific use cases

**Composition Strategies**:
- **Sequential Composition**: Chain agents in processing pipelines
- **Parallel Composition**: Run multiple agents simultaneously
- **Conditional Composition**: Select agents based on runtime conditions
- **Hierarchical Composition**: Nest agents within other agent workflows

## Reusability Patterns

### **Microservice Architecture**
```apl
// Create specialized agents for specific functions
AgentID = "authenticationAgent"
Description = "Handles user authentication and authorization"
Version = "1.0.0"

// Use in larger workflows
Workflows {
    "authentication-agent-1.0.0.kdeps"
}
```

### **Template-Based Development**
```apl
// Base template for API agents
AgentID = "apiTemplate"
Settings {
    APIServerMode = true
    APIServer {
        HostIP = "127.0.0.1"
        PortNum = 3000
    }
}

// Extend template for specific use cases
extends = "apiTemplate"
AgentID = "weatherApiAgent"
// Add specific weather-related resources
```

### **Resource Library Pattern**
```apl
// Create reusable resource libraries
// library/validation.pkl
ActionID = "emailValidation"
Category = "validation"
Run {
    Validation {
        Schema = emailSchema
        Data = "@(request.data().email)"
    }
}

// Use across multiple agents
Requires { "library.emailValidation" }
```

### **Plugin Architecture**
```apl
// Core agent with plugin support
AgentID = "coreAgent"
PluginInterface = "standardInterface"

// Plugins extend functionality
AgentID = "analyticsPlugin"
Implements = "standardInterface"
Category = "analytics"
```

## Advanced Composition Techniques

### **Agent Orchestration**
```apl
// Orchestrator agent that manages other agents
AgentID = "orchestratorAgent"
Description = "Manages complex multi-agent workflows"
Workflows {
    "data-processor-1.0.0.kdeps"
    "ml-analyzer-2.1.0.kdeps"
    "report-generator-1.5.0.kdeps"
}

Settings {
    OrchestrationMode = true
    AgentCoordination {
        Sequential = ["data-processor", "ml-analyzer"]
        Parallel = ["report-generator"]
        Conditional = {
            "premium-features": "@(request.data().tier == 'premium')"
        }
    }
}
```

### **Dynamic Agent Loading**
```apl
// Load agents based on runtime conditions
Expr {
    "local agentType = @(request.data().processingType)"
    "local agentPath = 'agents/' + agentType + '-agent.kdeps'"
    "@(runtime.loadAgent(agentPath))"
}
```

### **Agent Versioning and Migration**
```apl
// Handle multiple agent versions
Workflows {
    "data-processor-1.0.0.kdeps"  // Stable version
    "data-processor-2.0.0-beta.kdeps"  // Beta version
}

VersionStrategy {
    Default = "1.0.0"
    Beta = "@(request.headers().X-Beta-Features == 'true')"
    Rollback = "1.0.0"  // Fallback version
}
```

## Configuration Inheritance

### **Base Configuration Pattern**
```apl
// base-config.pkl - Common settings
BaseSettings {
    Environment = "production"
    RateLimitMax = 100
    AgentSettings {
        OllamaVersion = "0.8.0"
        Models { "llama3.2:1b" }
    }
}

// specific-agent.pkl - Inherits base settings
include "base-config.pkl"
AgentID = "specificAgent"
Settings = BaseSettings + {
    // Override or extend specific settings
    RateLimitMax = 200
    APIServerMode = true
}
```

### **Environment-Specific Configurations**
```apl
// development.pkl
DevSettings {
    Environment = "development"
    APIServer {
        PortNum = 3001
        CORS.AllowOrigins { "*" }
    }
    AgentSettings {
        Models { "tinydolphin" }  // Lightweight model for dev
    }
}

// production.pkl
ProdSettings {
    Environment = "production"
    APIServer {
        PortNum = 80
        CORS.AllowOrigins { "https://myapp.com" }
    }
    AgentSettings {
        Models { "llama3.2:3b" }  // Production model
    }
}
```

## Resource Sharing Strategies

### **Shared Resource Libraries**
```apl
// Create shared resource packages
// shared-resources.kdeps contains:
// - common validation resources
// - utility functions
// - standard response formats

// Use in multiple agents
AgentID = "myAgent"
Dependencies {
    "shared-resources-1.0.0.kdeps"
}
```

### **Resource Inheritance**
```apl
// Base resource definition
// base-llm.pkl
ActionID = "baseLLMResource"
Category = "ai"
Run {
    Chat {
        Model = "@(environment.get('DEFAULT_MODEL'))"
        TimeoutDuration = 60.s
        JSONResponse = true
    }
}

// Specialized resource
// specialized-llm.pkl
extends = "baseLLMResource"
ActionID = "specializedLLMResource"
Run {
    Chat = parent.Chat + {
        Prompt = "You are a specialized assistant for @(request.data().domain)"
        JSONResponseKeys { "answer"; "confidence"; "sources" }
    }
}
```

## Distribution and Packaging

### **Agent Packaging Best Practices**
```bash
# Create distributable packages
kdeps package my-agent --include-dependencies
kdeps package my-agent --optimize --compress

# Version management
kdeps package my-agent --version 1.2.0
kdeps package my-agent --tag latest
```

### **Package Management**
```bash
# Install shared agents
kdeps install authentication-agent-1.0.0.kdeps
kdeps install data-processor@2.1.0

# Update dependencies
kdeps update --check-compatibility
kdeps update authentication-agent --to 1.1.0
```

### **Registry and Distribution**
```bash
# Publish to registry
kdeps publish my-agent-1.0.0.kdeps --registry https://registry.kdeps.com

# Install from registry
kdeps install @company/authentication-agent
kdeps install @community/ml-utils
```

## Testing Reusable Components

### **Component Testing**
```apl
// test-config.pkl - Testing configuration
TestSettings {
    Environment = "test"
    MockMode = true
    TestData {
        SampleRequests { /* test data */ }
        ExpectedResponses { /* expected results */ }
    }
}
```

### **Integration Testing**
```bash
# Test agent composition
kdeps test my-composed-agent --with-dependencies
kdeps test --integration --agents agent1,agent2,agent3

# Performance testing
kdeps benchmark my-agent --requests 1000 --concurrent 10
```

## Best Practices

### **Design Principles**
- **Single Responsibility**: Each agent should have a clear, focused purpose
- **Loose Coupling**: Minimize dependencies between agents
- **High Cohesion**: Related functionality should be grouped together
- **Interface Consistency**: Use standard interfaces for interoperability

### **Versioning Strategy**
- Use semantic versioning (SemVer) for all agents and resources
- Maintain backward compatibility when possible
- Provide migration guides for breaking changes
- Test compatibility between different versions

### **Documentation Standards**
- Document all public interfaces and APIs
- Provide usage examples and integration guides
- Include performance characteristics and limitations
- Maintain change logs and version history

### **Security Considerations**
- Validate all inputs from external agents
- Implement proper access controls for shared resources
- Use secure communication channels for agent coordination
- Regular security audits for shared components

## Common Patterns and Examples

### **Multi-Tenant SaaS Architecture**
```apl
// Tenant-specific agent composition
AgentID = "tenantAgent"
Workflows {
    "core-platform-1.0.0.kdeps"        // Base platform
    "tenant-customization-@(tenant.id).kdeps"  // Tenant-specific
}

Settings {
    TenantIsolation = true
    DataPartitioning = "tenant"
}
```

### **Microservices Integration**
```apl
// Service composition for complex workflows
AgentID = "orderProcessingAgent"
Workflows {
    "user-service-1.0.0.kdeps"
    "inventory-service-2.0.0.kdeps"
    "payment-service-1.5.0.kdeps"
    "notification-service-1.0.0.kdeps"
}

Settings {
    ServiceMesh = true
    FailureHandling = "graceful"
}
```

### **Plugin Ecosystem**
```apl
// Core platform with plugin support
AgentID = "platformCore"
PluginDirectory = "plugins/"
SupportedPlugins {
    "analytics": "plugins/analytics-*.kdeps"
    "security": "plugins/security-*.kdeps"
    "ui": "plugins/ui-*.kdeps"
}
```

## Next Steps

- **[Tutorials](../tutorials/README.md)**: See practical examples of agent composition
- **[Core Resources](../core-resources/README.md)**: Understand the building blocks for reusable components
- **[Advanced Resources](../advanced-resources/README.md)**: Explore specialized capabilities for complex compositions

See the [remix.md](./remix.md) documentation for detailed examples and implementation patterns for agent composition and workflow sharing. 