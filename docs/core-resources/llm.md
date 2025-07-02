---
outline: deep
---

# LLM Resource

The `llm` resource facilitates the creation of a Large Language Model (LLM) session to interact with AI models effectively.

Multiple LLM models can be declared and used across multiple LLM resources. For more information, see the [Workflow](../configuration/workflow.md) documentation.

## Creating a New LLM Resource

To create a new `llm` chat resource, you can either generate a new AI agent using the `kdeps new` command or scaffold the resource directly.

Here's how to scaffold an `llm` resource:

``` bash
kdeps scaffold [aiagent] llm
```

This command will add an `llm` resource to the `aiagent/resources` folder, generating the following folder structure:

``` bash
aiagent
└── resources
    └── llm.pkl
```

The file includes essential metadata and common configurations, such as [Skip Conditions](../resources/skip) and [Preflight Validations](../resources/validations). For more details, refer to the [Common Resource Configurations](../resources/resources#common-resource-configurations) documentation.

## LLM Resource Structure

A complete LLM resource file looks like this:

``` apl
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
        // Chat configuration details
    }
}
```

## Chat Block

Within the `Run` block, you'll find the `Chat` block, structured as follows:

``` apl
Chat {
    Model = "tinydolphin" // Specifies the LLM model to use, defined in the workflow.

    // Send the dedicated prompt and role to the LLM or utilize the scenario block.
    // Specifies the LLM role context for this prompt, e.g., "user", "assistant", or "system".
    // Defaults to "human" if no role is specified.
    Role = "user"
    Prompt = "Who is @(request.data())?"

    // Scenario block allows adding multiple prompts and roles for this LLM session.
    scenario {
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

    // Determines if the LLM response should be a structured JSON.
    JSONResponse = true

    // If JSONResponse is true, the structured JSON will include the following keys:
    JSONResponseKeys {
        "first_name"
        "last_name"
        "parents"
        "address"
        "famous_quotes"
        "known_for"
    }

    // Specify the files that this LLM will process.
    files {
        // "@(request.files()[0])"
    }

    // Timeout duration in seconds, specifying when to terminate the LLM session.
    TimeoutDuration = 60.s
}
```

### Key Elements of the `Chat` Block

- **`Model`**: Specifies the LLM model to be used, as defined in the workflow configuration.
- **`Role`**: Defines the role context for the prompt, such as `user`, `assistant`, or `system`. Defaults to `human` if not specified.
- **`Prompt`**: The input query sent to the LLM for processing.
- **`Tools`**: Available tools for open-source LLMs to automatically use. See [Tools](../resources/tools) for more details.
- **`scenario`**: Enables the inclusion of multiple prompts and roles to shape the LLM session's context. Each `new` block within `scenario` specifies a role (e.g., `assistant` or `system`) and a corresponding prompt to guide the LLM's behavior or response.
- **`files`**: Lists files to be processed by the LLM, particularly useful for vision-based LLM models.
- **`JSONResponse`**: Indicates whether the LLM response should be formatted as structured JSON.
- **`JSONResponseKeys`**: Lists the required keys for the structured JSON response. Keys can include type annotations (e.g., `first_name__string`, `famous_quotes__array`, `details__markdown`, `age__integer`) to enforce specific data types.
- **`TimeoutDuration`**: Sets the execution timeout (e.g., in seconds `s` or minutes `min`), after which the LLM session is terminated.

When the resource is executed, you can leverage LLM functions like `llm.response("id")` to retrieve the generated response. For further details, refer to the [LLM Functions](../resources/functions.md#llm-resource-functions) documentation.

## Advanced Configuration

### Scenario Block Usage

The `scenario` block is particularly useful for setting up complex interactions with the LLM. By defining multiple roles and prompts, you can create a conversational context that guides the LLM's responses. For example:

``` apl
scenario {
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
        Prompt = "The Renaissance was a cultural movement that spanned roughly from the 14th to the 17th century..."
    }
}
```

This setup allows the LLM to maintain a consistent context across multiple interactions, improving response coherence.

### Handling Files

The `files` block supports processing of various file types, such as images or documents, which is particularly beneficial for multimodal LLMs. For example:

``` apl
files {
    "@(request.files()[0])" // Processes the first uploaded file
    "data/document.pdf"     // Processes a specific PDF file
}
```

Ensure that the files are accessible within the resource's context and compatible with the LLM model's capabilities.

### Structured JSON Responses

When `JSONResponse` is set to `true`, the LLM response is formatted as a JSON object with the keys specified in `JSONResponseKeys`. Type annotations can be used to enforce data types, ensuring the output meets specific requirements. For example:

``` apl
JSONResponseKeys {
    "name__string"
    "age__integer"
    "quotes__array"
    "bio__markdown"
}
```

This configuration ensures that the response contains a `name` (string), `age` (integer), `quotes` (array), and `bio` (markdown-formatted text).

### Tools Configuration

The `Tools` block allows open-source LLMs to utilize external tools to enhance their functionality, such as querying databases or executing scripts. Each tool is defined within a `new` block, specifying its name, script, description, and parameters. Tools can be chained, where the output of one tool is used as the input parameters for the next tool in the sequence.

For example:

``` apl
Tools {
    new {
        Name = "lookup_db"
        Script = "@(data.filepath("tools/1.0.0", "lookup.py"))"
        Description = "Lookup information in the DB"
        parameters {
            ["keyword"] { required = true; type = "string"; Description = "The string keyword to query the DB" }
        }
    }
    new {
        Name = "process_results"
        Script = "@(data.filepath("tools/1.0.0", "process.py"))"
        Description = "Process DB lookup results"
        parameters {
            ["lookup_data"] { required = true; type = "object"; Description = "The output data from lookup_db tool" }
        }
    }
}
```

#### Key Elements of the `Tools` Block

- **`Name`**: A unique identifier for the tool, used by the LLM to reference it.
- **`script`**: The path to the script or executable that the tool runs, often using a dynamic filepath like `@(data.filepath("tools/1.0.0", "lookup.py"))`.
- **`Description`**: A clear description of the tool's purpose, helping the LLM decide when to use it.
- **`Parameters`**: Defines the input parameters the tool accepts, including:
  - `Required`: Whether the parameter is mandatory (`true` or `false`).
  - `Type`: The data type of the parameter (e.g., `string`, `integer`, `object`, `boolean`).
  - `Description`: A brief explanation of the parameter's purpose.

#### Tool Chaining

Tools can be chained to create a pipeline where the output of one tool serves as the input for the next. The LLM automatically passes the output of a tool as the parameters for the subsequent tool, based on the order defined in the `Tools` block. For instance, in the example above, the `lookup_db` tool's output (e.g., a JSON object containing query results) is passed as the `lookup_data` parameter to the `process_results` tool.

To enable chaining:
- Ensure the output format of the first tool matches the expected input parameter type of the next tool (e.g., `object` for JSON data).
- Define tools in the order of execution, as the LLM processes them sequentially.
- Use clear `Description` fields to guide the LLM on when to initiate the chain.

#### Best Practices for Tools

- **Clear Descriptions**: Provide detailed descriptions to ensure the LLM understands when and how to use each tool.
- **Parameter Validation**: Specify parameter types and requirements to prevent errors, especially when chaining tools.
- **Script Accessibility**: Verify that script paths are correct and accessible within the resource's context.
- **Minimal Tools**: Include only necessary tools to avoid complexity, and order them logically for chaining.
- **Chaining Compatibility**: Ensure the output of one tool aligns with the input requirements of the next, using consistent data types.

For example, a chained weather data pipeline might look like:

``` apl
Tools {
    new {
        Name = "get_weather"
        Script = "@(data.filepath("tools/1.0.0", "weather.py"))"
        Description = "Fetches current weather data for a location"
        parameters {
            ["location"] { required = true; type = "string"; Description = "The city or region to fetch weather for" }
            ["unit"] { required = false; type = "string"; Description = "Temperature unit (e.g., Celsius or Fahrenheit)" }
        }
    }
    new {
        Name = "format_weather"
        Script = "@(data.filepath("tools/1.0.0", "format_weather.py"))"
        Description = "Formats weather data into a user-friendly summary"
        parameters {
            ["weather_data"] { required = true; type = "object"; Description = "The weather data from get_weather tool" }
        }
    }
}
```

For more information on tools, see [Tools](../resources/tools).

## Error Handling and Timeouts

The `TimeoutDuration` parameter is critical for managing long-running LLM sessions. If the session exceeds the specified duration (e.g., `60.s`), it will be terminated to prevent resource overuse. Ensure the timeout is set appropriately based on the complexity of the prompt and the model's performance.

Additionally, you can implement error handling using [Preflight Validations](../resources/validations) to check for valid inputs or model availability before executing the session.

## Best Practices

- **Model Selection**: Choose an LLM model that aligns with your use case (e.g., text generation, image processing) and is defined in the workflow configuration.
- **Prompt Design**: Craft clear and specific prompts to improve the quality of LLM responses. Use the `scenario` block to provide additional context where needed.
- **File Management**: Verify that files listed in the `files` block are accessible and compatible with the LLM model.
- **Structured Outputs**: Use `JSONResponse` and `JSONResponseKeys` for applications requiring structured data, and validate the output format in downstream processes.
- **Timeout Configuration**: Set a reasonable `TimeoutDuration` to balance performance and resource usage, especially for complex queries.
- **Tool Usage**: Configure tools with precise descriptions and parameters to ensure the LLM uses them effectively when needed.

## Example Use Case

Suppose you want to create an LLM resource to retrieve structured information about a history figure based on user input, with a tool to query a database for additional details. The complete resource might look like this:

``` apl
ActionID = "historyLLMResource"
Name = "Historical Figure Analyzer"
Description = "Provides structured information about historical figures"
Category = "ai"
Run {
    RestrictToHTTPMethods { "POST" }
    RestrictToRoutes { "/api/v1/history" }
    AllowedParams { "person" }
    PreflightCheck {
        Validations { "@(request.data('person'))" != "" }
        Retry = false
        RetryTimes = 1
    }
    PostflightCheck {
        Validations { "@(llm.response('historyLLMResource').name)" != "" }
        Retry = true
        RetryTimes = 2
    }
    Chat {
        Model = "tinydolphin"
        Role = "user"
        Prompt = "Provide details about @(request.data('person'))"
        scenario {
            new {
                Role = "assistant"
                Prompt = "You are a history expert AI, providing accurate and concise information about historical figures."
            }
        }
        Tools {
            new {
                Name = "lookup_db"
                Script = "@(data.filepath("tools/1.0.0", "lookup.py"))"
                Description = "Lookup historical figure details in the database"
                parameters {
                    ["name"] { required = true; type = "string"; Description = "The name of the historical figure" }
                }
            }
        }
        JSONResponse = true
        JSONResponseKeys {
            "name__string"
            "birth_year__integer"
            "known_for__array"
            "biography__markdown"
        }
        TimeoutDuration = 30.s
    }
}
```

This configuration ensures that the LLM returns a structured JSON response with details about the requested historical
figure, formatted according to the specified keys and types, and can use the `lookup_db` tool to fetch additional data
if needed.

For more advanced configurations and use cases, refer to the [Workflow](../configuration/workflow.md) and [LLM
Functions](../resources/functions.md#llm-resource-functions) documentation.
