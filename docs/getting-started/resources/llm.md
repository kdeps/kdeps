---
outline: deep
---

# LLM Resource

The `llm` resource facilitates the creation of a Large Language Model (LLM) session to interact with AI models effectively.

Multiple LLM models can be declared and used across multiple LLM resource. For more information, see the
[Workflow](../configuration/workflow.md) documentation.

## Creating a New LLM Resource

To create a new `llm` chat resource, you can either generate a new AI agent using the `kdeps new` command or scaffold the resource directly.

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

The file includes essential metadata and common configurations, such as [Skip Conditions](../resources/skip) and [Preflight Validations](../resources/validations). For more details, refer to the [Common Resource Configurations](../resources/resources#common-resource-configurations) documentation.

## Chat Block

Within the file, you’ll find the `chat` block, structured as follows:

```apl
chat {
    model = "tinydolphin" // Specifies the LLM model to use, defined in the workflow.
    prompt = "Who is @(request.data())?"

    // Determines if the LLM response should be a structured JSON.
    jsonResponse = true

    // If jsonResponse is true, the structured JSON will include the following keys:
    jsonResponseKeys {
        "first_name"
        "last_name"
        "parents"
        "address"
        "famous_quotes"
        "known_for"
    }

    // Timeout duration in seconds, specifying when to terminate the LLM session.
    timeoutSeconds = 60
}
```

Key Elements of the `chat` Block

- **`model`**: Specifies the LLM model to be used.
- **`prompt`**: The input prompt sent to the model.
- **`jsonResponse`**: Indicates if the response should be structured as JSON.
- **`jsonResponseKeys`**: Lists the required keys for the structured JSON response. To ensure the output conforms to
  specific data types, you can define the keys with their corresponding types. For example: `first_name__string`,
  `famous_quotes__array`, `details__markdown`, or `age__integer`.
- **`timeoutSeconds`**: Sets the execution timeout in seconds, after which the session is terminated.

When the resource is executed, you can leverage LLM functions like `llm.response("id")` to retrieve the generated
response. For further details, refer to the [LLM Functions](../resources/functions.md#llm-resource-functions)
documentation.
