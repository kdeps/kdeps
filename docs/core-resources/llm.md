---
outline: deep
---

# LLM Resource

The `llm` resource facilitates the creation of Large Language Model (LLM) sessions to interact with AI models effectively. This resource enables chat-based interactions, tool calling, structured JSON responses, and multimodal processing.

Multiple LLM models can be declared and used across multiple LLM resources. For model configuration details, see the [Workflow Configuration](../getting-started/configuration/workflow.md) documentation.

## Creating a New LLM Resource

To create a new `llm` chat resource, you can either generate a new AI agent using the `kdeps new` command or scaffold the resource directly.

**Scaffolding an LLM Resource:**
```bash
kdeps scaffold [aiagent] llm
```

This command adds an `llm` resource to the `aiagent/resources` folder, generating the following structure:

```bash
aiagent
└── resources
    └── llm.pkl
```

The generated file includes essential metadata and common configurations, such as [Skip Conditions](../workflow-control/skip.md) and [API Validations](../workflow-control/api-request-validations.md). For more details, refer to the [Resource Configuration](../resources.md) documentation.

## Basic LLM Resource Structure

A complete LLM resource file looks like this:

```apl
amends "resource.pkl"

ActionID = "llmResource"
Name = "AI Chat Handler" 
Description = "Processes user queries using LLM models"
Category = "ai"
Requires { "dataResource" }
Run {
    RestrictToHTTPMethods { "POST" }
    RestrictToRoutes { "/api/v1/chat" }
    AllowedHeaders { "Authorization"; "Content-Type" }
    AllowedParams { "query"; "model" }
    PreflightCheck {
        Validations { "@(request.data().query)" != "" }
        Retry = false
        RetryTimes = 1
    }
    PostflightCheck {
        Validations { "@(llm.response('llmResource').response)" != "" }
        Retry = true
        RetryTimes = 2
    }
    Chat {
        Model = "tinydolphin"
        Role = "user"
        Prompt = "Who is @(request.data())?"
        JSONResponse = false
        TimeoutDuration = 60.s
    }
}
```

## Chat Configuration Block

Within the `Run` block, the `Chat` block defines the LLM interaction parameters:

```apl
Chat {
    Model = "tinydolphin"
    Role = "user"
    Prompt = "Who is @(request.data())?"
    
    Scenario {
        new {
            Role = "assistant"
            Prompt = "You are a knowledgeable and supportive AI assistant with expertise in general information."
        }
        new {
            Role = "system"
            Prompt = "Ensure responses are concise and accurate, prioritizing user satisfaction."
        }
        new {
            Role = "system"
            Prompt = "If you are unsure and will just hallucinate your response, just lookup the DB"
        }
    }

    Tools {
        new {
            Name = "lookup_db"
            Script = "@(data.filepath("tools/1.0.0", "lookup.py"))"
            Description = "Lookup information in the DB"
            Parameters {
                ["keyword"] { Required = true; Type = "string"; Description = "The string keyword to query the DB" }
            }
        }
    }

    JSONResponse = true
    JSONResponseKeys {
        "first_name"
        "last_name"
        "parents"
        "address"
        "famous_quotes"
        "known_for"
    }

    Files {
        "@(request.files()[0])"
    }

    TimeoutDuration = 60.s
}
```

### Core Chat Properties

- **`Model`**: Specifies the LLM model to use, as defined in the workflow configuration (e.g., `"tinydolphin"`, `"llama3.3"`, `"llama3.2-vision"`)
- **`Role`**: Defines the role context for the prompt (`"user"`, `"assistant"`, or `"system"`). Defaults to `"user"` if not specified
- **`Prompt`**: The primary input query sent to the LLM for processing
- **`TimeoutDuration`**: Sets the execution timeout (e.g., `60.s`, `5.min`), after which the LLM session is terminated

### Advanced Chat Properties

- **`Scenario`**: Enables multiple prompts and roles to shape the LLM session's context
- **`Tools`**: Available tools for open-source LLMs to automatically use. See [Advanced Tools](../advanced-resources/tools.md) for details
- **`Files`**: Lists files to be processed by the LLM, particularly useful for vision-based models
- **`JSONResponse`**: Indicates whether the LLM response should be formatted as structured JSON
- **`JSONResponseKeys`**: Lists the required keys for structured JSON responses with optional type annotations

## Scenario Configuration

The `Scenario` block enables complex conversational contexts by defining multiple roles and prompts. This is particularly useful for setting up the LLM's personality, knowledge base, and behavioral guidelines.

### Basic Scenario Setup

```apl
Scenario {
    new {
        Role = "system"
        Prompt = "You are an expert in historical facts and provide detailed, accurate information."
    }
    new {
        Role = "user"
        Prompt = "Tell me about the Renaissance period."
    }
    new {
        Role = "assistant"
        Prompt = "The Renaissance was a cultural movement that spanned roughly from the 14th to the 17th century, beginning in Italy and later spreading across Europe..."
    }
}
```

### Advanced Scenario Patterns

**Knowledge Expert Setup:**
```apl
Scenario {
    new {
        Role = "system"
        Prompt = "You are a knowledgeable expert in @(request.data('domain')). Provide accurate, detailed information."
    }
    new {
        Role = "system"
        Prompt = "If you're uncertain about any information, use the available tools to look up accurate data."
    }
    new {
        Role = "system"
        Prompt = "Format your responses to be clear and structured for @(request.data('audience_level')) audience."
    }
}
```

**Conversational Agent Setup:**
```apl
Scenario {
    new {
        Role = "system"
        Prompt = "You are a helpful customer service agent for @(request.data('company_name'))."
    }
    new {
        Role = "system"
        Prompt = "Be polite, empathetic, and solution-focused. Always try to resolve customer issues."
    }
    new {
        Role = "assistant"
        Prompt = "Hello! I'm here to help you with any questions or concerns you may have."
    }
}
```

## File Processing

The `Files` block supports processing various file types, enabling multimodal interactions with vision-capable LLMs.

### File Input Methods

```apl
Files {
    "@(request.files()[0])"                    // First uploaded file
    "@(request.files('document'))"             // Named file input
    "@(data.filepath('uploads/image.jpg'))"   // Static file path
    "@(memory.get('processed_document'))"      // File from memory
}
```

### Supported File Types

- **Images**: JPG, PNG, WEBP, GIF (for vision models)
- **Documents**: PDF, TXT, DOCX (for document analysis)
- **Data Files**: JSON, CSV, XML (for data processing)
- **Code Files**: Various programming languages (for code analysis)

### File Processing Best Practices

```apl
Files {
    "@(request.files()[0])"
}

// Use with vision models for image analysis
Model = "llama3.2-vision"
Prompt = "Analyze this image and describe what you see: @(request.files()[0].description())"

// File validation in PreflightCheck
PreflightCheck {
    Validations { 
        "@(request.files().length())" > 0
        "@(request.files()[0].size())" < 10485760  // 10MB limit
    }
}
```

## Structured JSON Responses

When `JSONResponse` is set to `true`, the LLM response is formatted as a JSON object with keys specified in `JSONResponseKeys`.

### Basic JSON Configuration

```apl
JSONResponse = true
JSONResponseKeys {
    "name"
    "age"
    "occupation"
    "summary"
}
```

### Type-Annotated JSON Keys

Type annotations ensure specific data types in the response:

```apl
JSONResponseKeys {
    "name__string"           // String value
    "age__integer"          // Integer value
    "skills__array"         // Array of values
    "bio__markdown"         // Markdown-formatted text
    "active__boolean"       // Boolean value
    "details__object"       // Nested object
    "score__float"          // Floating-point number
    "timestamp__datetime"   // DateTime value
}
```

### Complex JSON Response Example

```apl
Chat {
    Model = "llama3.3"
    Prompt = "Analyze the person mentioned in: @(request.data('query'))"
    
    JSONResponse = true
    JSONResponseKeys {
        "basic_info__object"
        "achievements__array"
        "timeline__array"
        "analysis__markdown"
        "confidence_score__float"
        "sources__array"
    }
}
```

## Tools Configuration

The `Tools` block enables LLMs to use external tools for enhanced functionality, such as database queries, API calls, or script execution.

### Basic Tool Definition

```apl
Tools {
    new {
        Name = "lookup_db"
        Script = "@(data.filepath("tools/1.0.0", "lookup.py"))"
        Description = "Lookup information in the database"
        Parameters {
            ["keyword"] { Required = true; Type = "string"; Description = "The search keyword" }
            ["limit"] { Required = false; Type = "integer"; Description = "Maximum results to return" }
        }
    }
}
```

### Tool Chaining

Tools can be chained to create processing pipelines where the output of one tool becomes the input for the next:

```apl
Tools {
    new {
        Name = "get_weather"
        Script = "@(data.filepath("tools/1.0.0", "weather.py"))"
        Description = "Fetches current weather data for a location"
        Parameters {
            ["location"] { Required = true; Type = "string"; Description = "City or region name" }
            ["units"] { Required = false; Type = "string"; Description = "Temperature units (celsius/fahrenheit)" }
        }
    }
    new {
        Name = "format_weather"
        Script = "@(data.filepath("tools/1.0.0", "format_weather.py"))"
        Description = "Formats weather data into user-friendly summary"
        Parameters {
            ["weather_data"] { Required = true; Type = "object"; Description = "Weather data from get_weather tool" }
            ["format_type"] { Required = false; Type = "string"; Description = "Output format (brief/detailed)" }
        }
    }
}
```

### Tool Parameter Types

- **`string`**: Text values
- **`integer`**: Whole numbers
- **`float`**: Decimal numbers
- **`boolean`**: True/false values
- **`object`**: JSON objects
- **`array`**: Lists of values

### Tool Best Practices

1. **Clear Descriptions**: Provide detailed descriptions for when and how to use each tool
2. **Parameter Validation**: Specify required parameters and types
3. **Error Handling**: Include error checking in tool scripts
4. **Performance**: Keep tools lightweight and efficient
5. **Security**: Validate all inputs and sanitize outputs

For comprehensive tool configuration, see [Tools Documentation](../advanced-resources/tools.md).

## Error Handling and Validation

### Preflight Validation

Validate inputs before LLM execution:

```apl
PreflightCheck {
    Validations { 
        "@(request.data('query'))" != ""
        "@(request.data('query').length())" > 5
        "@(request.data('query').length())" < 1000
    }
    Retry = false
    RetryTimes = 1
}
```

### Postflight Validation

Validate LLM outputs after execution:

```apl
PostflightCheck {
    Validations { 
        "@(llm.response('llmResource').response)" != ""
        "@(llm.response('llmResource').response.length())" > 10
    }
    Retry = true
    RetryTimes = 2
}
```

### Timeout Management

Configure appropriate timeouts based on task complexity:

```apl
TimeoutDuration = 30.s   // Quick responses
TimeoutDuration = 2.min  // Complex analysis
TimeoutDuration = 5.min  // Heavy file processing
```

## Accessing LLM Responses

Use LLM functions to retrieve and process responses:

```apl
// Get the complete response
@(llm.response('llmResource').response)

// Get specific JSON fields
@(llm.response('llmResource').name)
@(llm.response('llmResource').age)

// Get response metadata
@(llm.response('llmResource').model)
@(llm.response('llmResource').tokens_used)
@(llm.response('llmResource').processing_time)
```

For detailed LLM function reference, see [Functions & Utilities](../functions-utilities/functions.md).

## Complete Example: Customer Support Agent

Here's a comprehensive example of an LLM resource for customer support:

```apl
amends "resource.pkl"

ActionID = "customerSupportLLM"
Name = "Customer Support AI Agent"
Description = "Handles customer inquiries with tool access and structured responses"
Category = "customer-service"
Requires { "dataResource" }

Run {
    RestrictToHTTPMethods { "POST" }
    RestrictToRoutes { "/api/v1/support" }
    AllowedHeaders { "Authorization"; "Content-Type"; "X-Customer-ID" }
    AllowedParams { "message"; "customer_id"; "priority" }
    
    PreflightCheck {
        Validations { 
            "@(request.data('message'))" != ""
            "@(request.data('customer_id'))" != ""
            "@(request.data('message').length())" < 2000
        }
        Retry = false
        RetryTimes = 1
    }
    
    PostflightCheck {
        Validations { 
            "@(llm.response('customerSupportLLM').response)" != ""
            "@(llm.response('customerSupportLLM').resolution_status)" != ""
        }
        Retry = true
        RetryTimes = 2
    }
    
    Chat {
        Model = "llama3.3"
        Role = "user"
        Prompt = "Customer message: @(request.data('message')). Customer ID: @(request.data('customer_id'))"
        
        Scenario {
            new {
                Role = "system"
                Prompt = "You are a professional customer support agent for TechCorp. Be helpful, empathetic, and solution-focused."
            }
            new {
                Role = "system"
                Prompt = "Always try to resolve issues quickly. Use tools to look up customer information and order details when needed."
            }
            new {
                Role = "system"
                Prompt = "If you cannot resolve an issue, escalate it appropriately with clear reasoning."
            }
        }
        
        Tools {
            new {
                Name = "lookup_customer"
                Script = "@(data.filepath("tools/1.0.0", "customer_lookup.py"))"
                Description = "Look up customer information and order history"
                Parameters {
                    ["customer_id"] { Required = true; Type = "string"; Description = "Customer ID to look up" }
                }
            }
            new {
                Name = "create_ticket"
                Script = "@(data.filepath("tools/1.0.0", "create_ticket.py"))"
                Description = "Create a support ticket for escalation"
                Parameters {
                    ["customer_id"] { Required = true; Type = "string"; Description = "Customer ID" }
                    ["issue_type"] { Required = true; Type = "string"; Description = "Type of issue" }
                    ["description"] { Required = true; Type = "string"; Description = "Issue description" }
                    ["priority"] { Required = true; Type = "string"; Description = "Priority level" }
                }
            }
        }
        
        JSONResponse = true
        JSONResponseKeys {
            "response__string"
            "resolution_status__string"
            "next_steps__array"
            "escalation_needed__boolean"
            "ticket_id__string"
            "estimated_resolution_time__string"
        }
        
        TimeoutDuration = 90.s
    }
}
```

## Best Practices

### Model Selection
- **Text Generation**: Use models like `llama3.3`, `mistral:7b` for general text tasks
- **Vision Tasks**: Use `llama3.2-vision` for image analysis
- **Code Tasks**: Use `codellama` for programming-related queries
- **Development**: Use lightweight models like `tinydolphin` for testing

### Prompt Engineering
- Be specific and clear in prompts
- Use the `Scenario` block to provide context and guidelines
- Include examples in system prompts when helpful
- Test prompts thoroughly with different inputs

### Performance Optimization
- Set appropriate timeouts based on task complexity
- Use structured JSON responses when you need specific output formats
- Implement proper validation to avoid unnecessary LLM calls
- Consider model size vs. performance trade-offs

### Security Considerations
- Validate all user inputs in PreflightCheck
- Sanitize file uploads and check file types
- Implement rate limiting for API endpoints
- Use appropriate authentication and authorization

### Error Handling
- Implement comprehensive validation checks
- Use retry logic for transient failures
- Provide meaningful error messages
- Log errors for debugging and monitoring

## Next Steps

- **[HTTP Client Resource](./client.md)**: Make external API calls from your LLM
- **[Python Resource](./python.md)**: Combine LLM with custom Python processing
- **[Response Resource](./response.md)**: Format and return LLM responses
- **[Advanced Tools](../advanced-resources/tools.md)**: Comprehensive tool configuration
- **[Functions & Utilities](../functions-utilities/functions.md)**: Available LLM functions

The LLM resource is the core of most AI agents in Kdeps. Master its configuration to build powerful, intelligent applications that can process natural language, handle complex queries, and integrate with external systems seamlessly.
