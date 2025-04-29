---
outline: deep
---

# LLM Resource

The `llm` resource facilitates the creation of a Large Language Model (LLM) session to interact with AI models
effectively.

Multiple LLM models can be declared and used across multiple LLM resources. For more information, see the
[Workflow](../configuration/workflow.md) documentation.


## Creating a New LLM Resource

To create a new `llm` chat resource, you can either generate a new AI agent using the `kdeps new` command or scaffold
the resource directly.

Here’s how to scaffold an `llm` resource:

```bash
kdeps scaffold [aiagent] llm
```

This command will add an `llm` resource to the `aiagent/resources` folder, generating the following folder structure:

```bash
aiagent
└── resources
    └── llm.pkl
```

The file includes essential metadata and common configurations, such as [Skip Conditions](../resources/skip) and
[Preflight Validations](../resources/validations). For more details, refer to the [Common Resource
Configurations](../resources/resources#common-resource-configurations) documentation.

## Chat Block

Within the file, you’ll find the `chat` block, structured as follows:

```apl
chat {
    model = "tinydolphin" // Specifies the LLM model to use, defined in the workflow.

    // Send the dedicated prompt and role to the LLM or utilize the scenario block.
    // Specifies the LLM role context for this prompt, e.g., "user", "assistant", or "system".
    // Defaults to "human" if no role is specified.
    role = "user"
    prompt = "Who is @(request.data())?"

    // Scenario block allows adding multiple prompts and roles for this LLM session.
    scenario {
        new {
            role = "assistant"
            prompt = "You are a knowledgeable and supportive AI assistant with expertise in general information."
        }
        new {
            role = "system"
            prompt = "Ensure responses are concise and accurate, prioritizing user satisfaction."
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
    timeoutDuration = 60.s
}
```

### Key Elements of the `chat` Block

- **`model`**: Specifies the LLM model to be used, as defined in the workflow configuration.
- **`role`**: Defines the role context for the prompt, such as `user`, `assistant`, or `system`. Defaults to `human` if not specified.
- **`prompt`**: The input query sent to the LLM for processing.
- **`scenario`**: Enables the inclusion of multiple prompts and roles to shape the LLM session's context. Each `new` block within `scenario` specifies a role (e.g., `assistant` or `system`) and a corresponding prompt to guide the LLM’s behavior or response.
- **`files`**: Lists files to be processed by the LLM, particularly useful for vision-based LLM models.
- **`JSONResponse`**: Indicates whether the LLM response should be formatted as structured JSON.
- **`JSONResponseKeys`**: Lists the required keys for the structured JSON response. Keys can include type annotations (e.g., `first_name__string`, `famous_quotes__array`, `details__markdown`, `age__integer`) to enforce specific data types.
- **`timeoutDuration`**: Sets the execution timeout (e.g., in seconds `s` or minutes `min`), after which the LLM session is terminated.

When the resource is executed, you can leverage LLM functions like `llm.response("id")` to retrieve the generated response. For further details, refer to the [LLM Functions](../resources/functions.md#llm-resource-functions) documentation.

## Advanced Configuration

### Scenario Block Usage

The `scenario` block is particularly useful for setting up complex interactions with the LLM. By defining multiple roles and prompts, you can create a conversational context that guides the LLM’s responses. For example:

```apl
scenario {
    new {
        role = "system"
        prompt = "You are an expert in historical facts and provide detailed, accurate information."
    }
    new {
        role = "user"
        prompt = "Tell me about the Renaissance period."
    }
    new {
        role = "assistant"
        prompt = "The Renaissance was a cultural movement that spanned roughly from the 14th to the 17th century..."
    }
}
```

This setup allows the LLM to maintain a consistent context across multiple interactions, improving response coherence.

### Handling Files

The `files` block supports processing of various file types, such as images or documents, which is particularly beneficial for multimodal LLMs. For example:

```apl
files {
    "@(request.files()[0])" // Processes the first uploaded file
    "data/document.pdf"     // Processes a specific PDF file
}
```

Ensure that the files are accessible within the resource’s context and compatible with the LLM model’s capabilities.

### Structured JSON Responses

When `JSONResponse` is set to `true`, the LLM response is formatted as a JSON object with the keys specified in `JSONResponseKeys`. Type annotations can be used to enforce data types, ensuring the output meets specific requirements. For example:

```apl
JSONResponseKeys {
    "name__string"
    "age__integer"
    "quotes__array"
    "bio__markdown"
}
```

This configuration ensures that the response contains a `name` (string), `age` (integer), `quotes` (array), and `bio` (markdown-formatted text).

## Error Handling and Timeouts

The `timeoutDuration` parameter is critical for managing long-running LLM sessions. If the session exceeds the specified duration (e.g., `60.s`), it will be terminated to prevent resource overuse. Ensure the timeout is set appropriately based on the complexity of the prompt and the model’s performance.

Additionally, you can implement error handling using [Preflight Validations](../resources/validations) to check for valid inputs or model availability before executing the session.

## Best Practices

- **Model Selection**: Choose an LLM model that aligns with your use case (e.g., text generation, image processing) and is defined in the workflow configuration.
- **Prompt Design**: Craft clear and specific prompts to improve the quality of LLM responses. Use the `scenario` block to provide additional context where needed.
- **File Management**: Verify that files listed in the `files` block are accessible and compatible with the LLM model.
- **Structured Outputs**: Use `JSONResponse` and `JSONResponseKeys` for applications requiring structured data, and validate the output format in downstream processes.
- **Timeout Configuration**: Set a reasonable `timeoutDuration` to balance performance and resource usage, especially for complex queries.

## Example Use Case

Suppose you want to create an LLM resource to retrieve structured information about a historical figure based on user input. The `chat` block might look like this:

```apl
chat {
    model = "tinydolphin"
    role = "user"
    prompt = "Provide details about @(request.data('person'))"
    scenario {
        new {
            role = "assistant"
            prompt = "You are a history expert AI, providing accurate and concise information about historical figures."
        }
    }
    JSONResponse = true
    JSONResponseKeys {
        "name__string"
        "birth_year__integer"
        "known_for__array"
        "biography__markdown"
    }
    timeoutDuration = 30.s
}
```

This configuration ensures that the LLM returns a structured JSON response with details about the requested historical figure, formatted according to the specified keys and types.

For more advanced configurations and use cases, refer to the [Workflow](../configuration/workflow.md) and [LLM Functions](../resources/functions.md#llm-resource-functions) documentation.
