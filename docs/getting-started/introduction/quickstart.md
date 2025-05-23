---
outline: deep
---

# Quickstart
To get started, make sure the Kdeps CLI is already [installed](./installation.md) on your device.

In this quickstart, we’ll guide you through creating a simple AI agent. We will:

1. **Enable API mode and set up a basic API route.**
2. **Calling an open-source LLM (Language Model) and passing request data for processing.**
3. **Returning a structured JSON response.**

This tutorial will walk you through building a foundational AI agent, which you can expand to handle more advanced
workflows. Let’s dive in!

## Generate the Configuration

Kdeps requires a `.kdeps.pkl` configuration file to function. This file is automatically generated the first time you
run `kdeps`, with default settings, including the GPU mode (`dockerGPU`) is set to `cpu`. Other `dockerGPU` values
includes `nvidia`, or `amd`.

To create the configuration file, simply execute:

```bash
kdeps
```
After generation, an editor defined in the `$EDITOR` env var will edit the configuration, which allows you to change the
`dockerGPU` to the GPU that you have.

<img alt="Kdeps - Configuration" src="/configuration.gif" />

This is a one-time setup step. For more information, see the [configuration guide](../configuration/configuration.md).

## Create Your First AI Agent

Kdeps can generate a starting project with the `new` command. To create a new project, use the following
command:

```bash
kdeps new aiagentx
```

This command will:

1. **Create a project named `aiagentx`.**
2. **Set up a project directory called `aiagentx`** with the following structure:
- **`workflow.pkl`**: A [Workflow](../configuration/workflow.md) configuration file that defines the workflow and settings for your AI agent project.
- **`resources/`**: A [Resources](../resources/resources.md) folder pre-filled with example
         [`.pkl`](https://pkl-lang.org/index.html) resource files, including:
     - **`resources/python.pkl`**: A [Python Resource](../resources/python.md) for running Python scripts.
     - **`resources/response.pkl`**: A [Response Resource](../resources/response.md) for preparing JSON responses from APIs.
     - **`resources/client.pkl`**: An [HTTP Client Resource](../resources/client.md) for making API requests.
     - **`resources/exec.pkl`**: A [Exec Resource](../resources/exec.md) for executing shell commands.
     - **`resources/llm.pkl`**: A [LLM Resource](../resources/llm.md) for interacting with language models (LLMs).
- **`data/`**: A [Data](../resources/data.md) directory for storing project-specific data.

Once the setup is complete, you’re ready to start building and customizing your AI agent.

> **Note:**
> In this quickstart guide, we are just going to focus on configuring the `workflow.pkl`, and 2 resources namely
> `resources/chat.pkl` and `resources/response.pkl`.

## Configure the Workflow

The `workflow.pkl` file is the core configuration for your AI agent. It allows you to define API routes, configure
custom Ubuntu repositories and packages, manage Anaconda and Python dependencies, and include LLM models. For
comprehensive details, see the [`Workflow`](../configuration/workflow.md) documentation.

### Default Action

The `workflow.pkl` file defines the workflow and settings for your AI agent. Within this file, you’ll find the `targetActionID`
configuration:

```apl
targetActionID = "responseResource"
```

Here, `responseResource` refers to the ID of the target resource file, located in `resources/response.pkl`:

```apl
actionID = "responseResource"
```

This resource will be executed as the default action whenever the AI agent runs.

### API Mode

The `workflow.pkl` file allows you to configure the AI agent to operate in API mode. Below is the generated
configuration:

```apl
APIServerMode = true
APIServer {
...
    portNum = 3000
    routes {
        new {
            path = "/api/v1/whois"
            methods {
                "GET" // Enables data retrieval
                "POST" // Allows data submission
            }
        }
    }
}
```

This configuration creates an API server running on port `3000` with a route at `/api/v1/whois`. You can define multiple
routes as needed.


With these settings, you can interact with the API using `curl` or similar tools. For example:

```bash
curl 'http://localhost:3000/api/v1/whois' -X GET
```

If you set `APIServerMode` to `false`, the AI agent will bypass the API server and directly execute the default action,
exiting upon completion.

### LLM Models

The `workflow.pkl` file defines the LLM models to be included in the Docker image. Here’s an example configuration:

```apl
agentSettings {
...
    models {
        "tinydolphin"
        // "llama3.2"
        // "llama3.1"
    }
}
```

In this setup, the `llama*` models are commented out. To enable them, simply uncomment the corresponding lines.

Kdeps uses [Ollama](https://ollama.com) as it's LLM backend. You can define as many Ollama compatible models as needed
to fit your use case.

For a comprehensive list of available Ollama compatible models, visit the [Ollama model
library](https://ollama.com/library).


> **Note: Additional settings about Ubuntu, Python, and Anaconda Packages**
>
> You can also configure custom Ubuntu packages, repositories, and PPAs, along with additional Python or Anaconda
> packages. However, for this quickstart guide, we won’t be using these settings.

## Configuring Resources

Once the `workflow.pkl` is configured, we can move on to setting up the resources. The AI agent will utilize two
resources: `resources/chat.pkl` and `resources/response.pkl`.

As mentioned previously, each resource is assigned a unique ID, which we will use to refer to the corresponding
resource.

### Resource Dependencies

Each resource contains a `requires` section that defines the dependencies needed for that resource. The generated
`resources/response.pkl` resource is dependent on the `chatResource` resource. Additionally, you can include other
resources by uncommenting the relevant lines.

```apl
requires {
    "chatResource"
    // "pythonResource"
    // "shellResource"
    // "httpResource"
}
```

When the resource IDs are specified, they form a dependency graph that determines the execution order of the
workflow. To learn more about how this graph-based dependency system works, refer to the [Graph
Dependency](../resources/kartographer.md) documentation.

### `resources/response.pkl` Resource

Within the `resources/response.pkl`, you'll find the following structure:

```apl
APIResponse {
    success = true
    response {
        data {
            "@(llm.response("chatResource"))"
            // "@(python.stdout("pythonResource"))"
            // "@(exec.stdout("shellResource"))"
            // "@(client.responseBody("httpResource"))"
        }
    }
    errors {
        new {
            code = 0
            message = ""
        }
    }
}
```

The `APIResponse` directive is the structure that will be converted into a JSON response.

The resulting JSON will generally look like this:

```json
{
    "success": true,
    "response": {
        "data": []
     },
    "errors": [{
        "code": 0,
        "message": ""
    }]
}
```

### Functions

Within the `data` JSON array, you will encounter references such as `llm`, `python`, `exec`, and `client`. These
represent resource functions, which are fully customizable using [Apple PKL](https://pkl-lang.org). This flexibility
allows you to extend and adapt the resources to meet your specific requirements. See the
[Functions](../resources/functions.md) documentation, for more information.

Each resource corresponds to a specific function, as illustrated below:

```apl
llm.response("ID")
// python.stdout("ID")
// exec.stdout("ID")
// client.responseBody("ID")
```

In this AI agent workflow, the LLM response is retrieved from the `chatResource` and appended to the `data` JSON array.

### Resource Promise

Notice that each resource function is enclosed within `"@()"`. This follows the Kdeps convention, which ensures the
resource is executed at a later stage. For more details on this convention, refer to the documentation on the
[Kdeps Promise](../resources/promise.md) directive.

When invoking a resource function, always wrap it in `"@()"` along with double quotes, as in
`"@(llm.response("chatResource"))"`. Depending on the output of this promise, you may sometimes needed to escape it.

For example:

```apl
local clientResponse =
"""
@(client.responseBody("ID"))
"""
```

### `resources/chat.pkl` Resource

The chat resource is an `llm` resource. This will create our LLM chat sessions.

If we look at the pkl file, we notice that the `requires` section is blank. This is because `chatResource` does not
depend on other resource in order to function.

```apl
chat {
    model = "llama3.1"
    prompt = "Who is @(request.data())?"
    JSONResponse = true
    JSONResponseKeys {
        "first_name"
        "last_name"
        "parents"
        "address"
        "famous_quotes"
        "known_for"
    }
    timeoutDuration = 60.s
}
```

The `model` we use here is the same model that we define in the `workflow.pkl`. If you want to use multiple LLMs. You
need to create new `llm` resource file to use the defined LLM model there.

In the `prompt`, we use the function `@(request.data())`, which inserts the request data into the prompt. Referring back
to the route configuration, the `curl` command can send request data using the `-d` flag, as shown:

```bash
curl 'http://localhost:3000/api/v1/whois' -X GET -d "Neil Armstrong"
```

Additionally, we have set `JSONResponse` to `true`, enabling the use of `JSONResponseKeys`. To ensure the output
conforms to specific data types, you can define the keys with their corresponding types. For example:
`first_name__string`, `famous_quotes__array`, `details__string`, or `age__integer`.

> **Important:**
> To accomplish defining the corresponding data types to keys, you'll need to adjust your LLM model, as the default
> `tinydolphin` model is not equipped to handle this. It is recommended to use models from the `llama3.*` family
> instead.

## Packaging

After finalizing our AI agent, we can then proceed on packaging the AI agent. Packaged AI agents are single file that
ends in `.kdeps` extension. With a single file, we can distribute it, reuse, sell and remix it in your AI agents.

To package an AI agent, simply run with `package` specifying the folder.

```bash
kdeps package aiagentx
```

This will create the `aiagentx-1.0.0.kdeps` file in the current directory. We can now proceed on building the Docker
image and container for this AI agent.

## Dockerization

After building the `kdeps` file, we can now run either `build` or `run` on the AI agent file.

The `build` command will create a Docker image, and the `run` will both create a Docker image and container.

On this example, we will use `run` to do both.

```bash
kdeps run aiagentx-1.0.0.kdeps
```

This step will build the image, install the necessary packages, and download the Ollama models.

## Testing the AI Agent API

After creating the container. the LLM models will be downloaded by Ollama. This might take some time, the API routes
will be not be available after Ollama have completed downloading the models.

After the models has been downloaded, we can proceed on doing an API call using `curl`.

```json
> curl 'http://localhost:3000/api/v1/whois' -X GET -d "Neil Armstrong"

{
  "errors": [
    {
      "code": 0,
      "message": ""
    }
  ],
  "response": {
    "data": [
      {
        "address": "Lebanon, Ohio, USA (birthplace)",
        "famous_quotes": [
          "That's one small step for man, one giant leap for mankind."
        ],
        "first_name": "Neil",
        "known_for": [
          "First person to walk on the Moon during the Apollo 11 mission",
          "Pioneering astronaut and naval aviator"
        ],
        "last_name": "Armstrong",
        "parents": {
          "father": "Stephen Koenig Armstrong",
          "mother": "Viola Louise Engel"
        }
      }
    ]
  },
  "success": true
}%

```

<img alt="Kdeps - API" src="/api.gif" />

And that's it! We have created our first Kdeps AI Agent. Now, let's dive into Kdeps configuration a bit further and
learn more about workflow and resources.
