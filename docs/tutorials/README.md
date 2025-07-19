# Tutorials

Practical, step-by-step tutorials and real-world examples for building AI agents with Kdeps. These tutorials provide hands-on experience with different aspects of the framework and demonstrate best practices for various use cases.

## Available Tutorials

### **[Weather API Tutorial (`how-to-weather-api.md`)](./how-to-weather-api.md)**
**Complete API Development Example**

Build a comprehensive weather information API that demonstrates fundamental Kdeps concepts. This tutorial covers:

- **API Design**: Creating RESTful endpoints for weather data
- **External Integration**: Connecting to third-party weather services
- **Data Processing**: Parsing and transforming weather data
- **LLM Integration**: Adding natural language weather descriptions
- **Error Handling**: Implementing robust error management
- **Response Formatting**: Creating structured JSON responses

**What You'll Learn**:
- HTTP Client resource configuration and usage
- API response resource design and implementation
- Request validation and data processing
- Integration with external APIs
- LLM-powered content enhancement

**Skills Developed**:
- API development patterns
- External service integration
- Data transformation techniques
- Error handling strategies

### **[Structured LLM Responses (`how-to-structure-llm.md`)](./how-to-structure-llm.md)**
**JSON Output Formatting and Schema Design**

Learn how to create LLM interactions that produce consistent, structured JSON outputs. This tutorial demonstrates:

- **Schema Design**: Creating robust JSON schemas for LLM outputs
- **Prompt Engineering**: Crafting prompts for structured responses
- **Type Enforcement**: Ensuring data type consistency
- **Validation**: Implementing output validation and error handling
- **Advanced Patterns**: Complex structured output scenarios

**What You'll Learn**:
- LLM configuration for structured output
- JSON schema design and validation
- Prompt engineering best practices
- Type system usage and enforcement
- Advanced LLM response patterns

**Skills Developed**:
- Structured AI output design
- Schema validation techniques
- Prompt optimization strategies
- Type-safe AI interactions

## Learning Paths

### **Beginner Path: Getting Started**
1. **[Quickstart Guide](../getting-started/introduction/quickstart.md)** - Build your first AI agent
2. **[Weather API Tutorial](./how-to-weather-api.md)** - Learn API development basics
3. **[Core Resources](../core-resources/README.md)** - Understand fundamental building blocks

**Prerequisites**: Basic understanding of APIs and JSON
**Time Commitment**: 2-3 hours
**Skills Gained**: Basic AI agent development, API integration, resource composition

### **Intermediate Path: Advanced Functionality**
1. **[Structured LLM Responses](./how-to-structure-llm.md)** - Master LLM output control
2. **[Workflow Control](../workflow-control/README.md)** - Add conditional logic and validation
3. **[Functions & Utilities](../functions-utilities/README.md)** - Leverage advanced utilities

**Prerequisites**: Completion of beginner path
**Time Commitment**: 3-4 hours
**Skills Gained**: Advanced LLM usage, workflow design, validation patterns

### **Advanced Path: Complex Systems**
1. **[Advanced Resources](../advanced-resources/README.md)** - Explore specialized capabilities
2. **[Data & Memory](../data-memory/README.md)** - Master data management
3. **[Reusability](../reusability/README.md)** - Build composable systems

**Prerequisites**: Completion of intermediate path
**Time Commitment**: 4-5 hours
**Skills Gained**: Complex system design, advanced AI features, enterprise patterns

## Tutorial Categories

### **API Development**
- **Weather API Tutorial**: External API integration and data processing
- **Authentication API**: User management and security patterns
- **File Upload API**: File handling and processing workflows
- **Real-time API**: WebSocket and streaming data patterns

### **AI Integration**
- **Structured LLM Responses**: Controlled AI output formatting
- **Multi-modal Processing**: Combining text and image AI capabilities
- **Tool-Enhanced AI**: Integrating external tools with LLM conversations
- **Conversation Management**: Building stateful chat applications

### **Data Processing**
- **ETL Workflows**: Data extraction, transformation, and loading
- **Batch Processing**: Handling large datasets efficiently
- **Real-time Analytics**: Processing streaming data with AI
- **Document Processing**: Automated document analysis and extraction

### **Enterprise Patterns**
- **Microservices Architecture**: Building distributed AI systems
- **Multi-tenant Applications**: SaaS patterns with AI agents
- **Scalability Patterns**: Designing for high-performance scenarios
- **Security Implementation**: Authentication, authorization, and data protection

## Practical Examples

### **Simple Weather API**
```apl
// Basic weather information endpoint
AgentID = "weatherAPI"
TargetActionID = "weatherResponse"
Settings {
    APIServerMode = true
    APIServer {
        Routes {
            new { Path = "/weather"; Method = "GET" }
        }
    }
}
```

### **AI-Powered Content Generator**
```apl
// Content generation with structured output
ActionID = "contentGenerator"
Run {
    Chat {
        Model = "llama3.2:1b"
        Prompt = "Generate a blog post about @(request.data().topic)"
        JSONResponse = true
        JSONResponseKeys {
            "title"
            "content"
            "tags"
            "summary"
        }
    }
}
```

### **File Processing Pipeline**
```apl
// Automated document processing
ActionID = "documentProcessor"
Requires { "uploadValidator"; "textExtractor"; "aiAnalyzer" }
Run {
    FileUpload {
        AllowedTypes { "pdf"; "docx"; "txt" }
        ProcessingPipeline = "documentAnalysis"
    }
}
```

## Best Practices Demonstrated

### **Error Handling**
All tutorials demonstrate proper error handling patterns:
- Input validation and sanitization
- Graceful degradation for external service failures
- Meaningful error messages and status codes
- Logging and monitoring integration

### **Performance Optimization**
Tutorials include performance considerations:
- Efficient resource usage and caching
- Parallel processing where appropriate
- Memory management for large datasets
- Response time optimization techniques

### **Security Implementation**
Security best practices are woven throughout:
- Input validation and sanitization
- Authentication and authorization patterns
- Secure external API integration
- Data protection and privacy considerations

### **Testing Strategies**
Each tutorial includes testing approaches:
- Unit testing for individual resources
- Integration testing for complete workflows
- Performance testing and benchmarking
- Error scenario testing

## Getting the Most from Tutorials

### **Before You Start**
- Ensure Kdeps is properly installed and configured
- Have a code editor ready for configuration files
- Understand basic JSON and API concepts
- Review relevant documentation sections

### **During the Tutorial**
- Follow examples step-by-step
- Experiment with variations and modifications
- Test each component as you build it
- Take notes on patterns and best practices

### **After Completion**
- Build on tutorial examples with your own ideas
- Combine patterns from different tutorials
- Contribute improvements or additional examples
- Share your implementations with the community

## Community Examples

### **Sample Projects**
- **E-commerce Assistant**: AI-powered shopping recommendations
- **Document Analyzer**: Automated document classification and extraction
- **Social Media Manager**: Content generation and scheduling automation
- **Customer Support Bot**: Intelligent ticket routing and response generation

### **Integration Examples**
- **Slack Bot Integration**: AI agents in team communication
- **CRM Enhancement**: AI-powered customer data analysis
- **Content Management**: Automated content creation and optimization
- **Data Pipeline Automation**: AI-enhanced ETL processes

## Contributing to Tutorials

### **Submission Guidelines**
- Follow established tutorial structure and format
- Include complete, working examples
- Provide clear step-by-step instructions
- Test all code examples thoroughly

### **Tutorial Topics Needed**
- Mobile app backend development
- Real-time data processing
- Machine learning model integration
- Advanced security patterns
- Performance optimization techniques

## Next Steps

1. **Choose Your Learning Path**: Select tutorials based on your experience level
2. **Set Up Environment**: Ensure Kdeps is properly configured
3. **Start with Basics**: Begin with fundamental concepts before advancing
4. **Practice and Experiment**: Modify examples to deepen understanding
5. **Build Real Projects**: Apply learned concepts to practical applications

## Additional Resources

- **[Installation Guide](../getting-started/introduction/installation.md)**: Set up your development environment
- **[Configuration Reference](../getting-started/configuration/configuration.md)**: Understand system configuration
- **[API Reference](../resources.md)**: Complete function and resource documentation
- **[Community Examples](https://github.com/kdeps/examples)**: More examples and community contributions

Start your Kdeps journey with the [Weather API Tutorial](./how-to-weather-api.md) for a comprehensive introduction to AI agent development! 