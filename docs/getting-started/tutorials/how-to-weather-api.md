---
outline: deep
---

# Building an AI-Assisted Weather Forecaster API

This tutorial demonstrates how to connect to an external API using the HTTP client resource, process the data with an
LLM, and deliver structured JSON outputs. Using the [Open-Meteo API](https://open-meteo.com/) as an example, we’ll walk
through creating consistent and reliable responses for integration into your applications.

The example will use the [`apiServeMode`](/getting-started/configuration/workflow#api-server-settings), but the same approach can be adapted for [Lambda Mode](/getting-started/configuration/workflow#lambda-mode).

## Getting Started

Let’s begin by setting up our project. This tutorial will guide you through scaffolding four essential `.pkl` files:

- **`workflow.pkl`**: Configures the AI Agent. See details in [Workflow](/getting-started/configuration/workflow).
- **`resources/llm.pkl`**: Defines the LLM model responsible for generating structured JSON output. Learn more in [LLM
  Resource](/getting-started/resources/functions#llm-resource-functions).
- **`resources/response.pkl`**: Handles structured JSON output responses. Check out [Response
  Resource](/getting-started/resources/response).
- **`resources/client.pkl`**: Creates an HTTP Client to call an API.
- **`resources/exec.pkl`**: Prepares and saves the LLM-generated output as a JSON file for subsequent processing.

By the end of this tutorial, you’ll have a foundational understanding of how to structure LLM responses for API-based
applications, and using the client resource to connect to an external API.

## Setting Up Our AI Agent: `weather_forecast_ai`

To begin, we will scaffold the resources for our project. We'll name our AI agent `weather_forecast_ai` and use the
following command to generate the necessary files all at once:

```bash
kdeps scaffold weather_forecast_ai workflow llm response client exec
```

This command will create the following directory structure:

```plaintext
weather_forecast_ai
├── data
├── resources
│   ├── client.pkl
│   ├── llm.pkl
│   ├── exec.pkl
│   └── response.pkl
└── workflow.pkl
```

### Updating the `workflow.pkl` File

Next, let's modify the `workflow.pkl` file by adding a new route to the `APIServer` block. Here's the updated configuration:

```diff
APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000

    routes {
        new {
            path = "/api/v1/forecast" // [!code ++]
            methods {
                "GET"                 // [!code ++]
            }
        }
    }
}
```

### Adding a Model to the Agent

Finally, we will include the `llama3.1` model in the `models` block within the `agentSettings` section:

```diff
agentSettings {
    ...
    models {
        "llama3.1"                   // [!code ++]
    }
    ...
}
```

## Creating Our First LLM for Structured Responses

In this step, we'll build our first Language Learning Model (LLM) to assist with preparing inputs for an external Weather API call. The model will transform natural language queries into structured data, enabling seamless interaction with the API.

### Renaming and Duplicating Files

First, let's rename the `resources/llm.pkl` file to `resources/llm_input.pkl`:

```bash
mv weather_forecast_ai/resources/llm.pkl \
   weather_forecast_ai/resources/llm_input.pkl
```

Next, create a duplicate of this file, naming it `resources/llm_output.pkl`. This will handle output preparation:

```bash
cp weather_forecast_ai/resources/llm_input.pkl \
   weather_forecast_ai/resources/llm_output.pkl
```

### Choosing the Weather API

We'll use [Open Mateo](https://open-mateo.com) for this project, as it offers free API access. If you use an API key,
you'll need to define it as `args` in the `workflow.pkl` file, and add the key to your `.env` file.

The following input data is required:

- **Latitude**
- **Longitude**
- **Timezone**

This data will be used to retrieve:

- Hourly and daily temperature
- Precipitation
- Wind speed

## Constructing the Input

We'll use natural language queries to obtain weather information and pass these as parameters to the API. The query will use the key `q` for the parameter. Here's an example API request:

```bash
http://localhost:3000/api/v1/forecast?q=What+is+the+weather+in+Amsterdam?
```

### Updating `llm_input.pkl`

Open the `resources/llm_input.pkl` file and update the resource details as follows:

```diff
actionID = "llmInput"                                            // [!code ++]
name = "AI Helper for Input"
description = "An AI helper to parse input into structured data"
```

### Adding Model, Prompt, and Response Keys

Next, define the model, prompt, and structured response keys:

```diff
chat {
    model = "llama3.1"                                    // [!code ++]
    prompt = """
Extract the longitude, latitude, and timezone             // [!code ++]
from this text. An example of timezone is Asia/Manila.    // [!code ++]
@(request.params("q"))?                                    // [!code ++]
"""
    JSONResponse = true                                   // [!code ++]
    JSONResponseKeys {
        "longitude_str"                                   // [!code ++]
        "latitude_str"                                    // [!code ++]
        "timezone_str"                                    // [!code ++]
    }
...
}
```

Key points:

- **`@(request.params("q"))`**: This function extracts the query parameter with the ID `q`.
- **Appended `_str` to Response Keys**: Adding `_str` enforces a typed structure for the LLM output.

## Storing the LLM response to a JSON file

After we have generated the structured LLM output from the request params, we need to store it to JSON to parse the
necessary data required for the HTTP Client resource.

Since each resource can only execute a single dedicated task per run, another resource is required to record the JSON
response to a file.

In this, we will use the `exec` resource.

### Editing `resources/exec.pkl`

First, update the `resources/exec.pkl` file as follows:

```diff
actionID = "execResource"                                       // [!code ++]
name = "Store LLM JSON response to a file"
description = "This resource will store the LLM JSON response to a file for processing later"
requires {
    "llmInput"                                            // [!code ++]
}
```

By defining `llmInput` as a dependency, this resource can access its outputs.

### Creating the JSON file

Let's say `/tmp/llm_input.json` is the name of the file, we need to create this file from the output of the `llmInput`.
We also ensure that this file is recreated by deleting it first, so that we are sure that it's a fresh file, not a
previously generated file that we might have reused.

```diff
exec {
    command = """
    rm -rf /tmp/llm_input.json
    echo $LLM_INPUT > /tmp/llm_input.json
    """
    env {
        ["LLM_INPUT"] = "@(llm.response("llmInput"))"
    }
```

## Creating an HTTP Client for the Weather API

In this step, we will build the HTTP client responsible for interacting with the Weather API. Using the structured
output (`longitude_str`, `latitude_str`, and `timezone_str`) from our previous LLM call, we’ll extract these values the
built-in JSON parser.

### Editing `resources/client.pkl`

First, update the `resources/client.pkl` file as follows:

```diff
actionID = "HTTPClient"                                                    // [!code ++]
name = "HTTP Client for the Weather API"
description = "This resource enables API requests to the Weather API."
requires {
    "execResource"                                                   // [!code ++]
}
```

### Defining the HTTP Client

Now, define the `HTTPClient` block to handle API calls:

```diff
HTTPClient {
    method = "GET"                                                             // [!code ++]
    url = "https://api.open-meteo.com/v1/forecast"                             // [!code ++]
    params {
        ["current_weather"] = "true"                                           // [!code ++]
        ["forecast_days"] = "1"                                                // [!code ++]
        ["hourly"] = "temperature_2m,precipitation,wind_speed_10m"             // [!code ++]
        ["daily"] = "temperature_2m_max,temperature_2m_min,precipitation_sum"  // [!code ++]
    }
...
```

This is pretty much straightforward, we created params for the API with the data that we need.

### Adding the JSON data

In the previous resource, we save the LLM output into the `/tmp/llm_input.json` file.

Here's how to define a `local` variable that contains the content of this file:

```diff
local JSONData = """
@(read?("file:/tmp/llm_input.json")?.text)
"""
```

Then let's add the remaining variables to the parameters, which we will parse from the JSON file.

```diff
local JSONData = """                                                           // [!code ++]
@(read?("file:/tmp/llm_input.json")?.text)                                     // [!code ++]
"""                                                                            // [!code ++]

HTTPClient {
    method = "GET"
    url = "https://api.open-meteo.com/v1/forecast"
    data {}
    params {
        ["latitude" ] = "@(JSONParser.parse(JSONData)?.latitude_str)"           // [!code ++]
        ["longitude"] = "@(JSONParser.parse(JSONData)?.longitude_str)"          // [!code ++]
        ["timezone "] = "@(JSONParser.parse(JSONData)?.timezone_str)"           // [!code ++]
        ["current_weather"] = "true"
        ["forecast_days"] = "1"
        ["hourly"] = "temperature_2m,precipitation,wind_speed_10m"
        ["daily"] = "temperature_2m_max,temperature_2m_min,precipitation_sum"
    }
```

We use the built-in `JSONParser.parse(file)` to obtain the three values from the JSON file.
The fields `latitude_str`, `longitude_str`, and `timezone_str` are parsed and added to the params.

> **Important:**
> Please note that we use the `?` between functions, which serves as the `null-safe` operator. During runtime, Kdeps
> will first parse all PKL files to gather file metadata, then build and post-process the configurations and graphs. To
> prevent the actual execution of the functions, ensure that the function call is `null-safe`.

## Constructing the Output Using LLM

With the JSON response from the Weather API in hand, let's format it into a user-friendly output using the LLM.

### Updating `llm_output.pkl`

Open the `resources/llm_output.pkl` file and update the resource details as follows:

```diff
actionID = "llmOutput"                                     // [!code ++]
name = "AI Helper for Output"
description = "A resource to generate a polished output using LLM."
requires {
    "HTTPClient"                                     // [!code ++]
}
```

### Adding the Output Formatting Logic

Next, configure the output construction logic:

```diff
chat {
    model = "llama3.1"                                                      // [!code ++]
    prompt = """
As if you're a weather reporter, present this response in an engaging way:  // [!code ++]
@(client.responseBody("HTTPClient").base64Decoded)                          // [!code ++]
"""
    JSONResponse = false                                                    // [!code ++]
...
```

- **`model`**: Specifies the LLM version (`llama3.1`).
- **`prompt`**: Instructs the model to generate an output resembling a weather reporter's announcement. The raw API
response is passed via `@(client.responseBody("HTTPClient"))`.
- **`JSONResponse = false`**: Indicates that the output should be plain text, formatted for readability, rather than
structured JSON.

This setup ensures that the output from the Weather API is transformed into an engaging and easy-to-understand
narrative, making it more relatable for end users.

## Finalizing the AI Agent API

To complete the AI agent, we’ll incorporate a `response` resource that enables the API to deliver structured JSON responses.

### Updating `resources/response.pkl`

Edit the `resources/response.pkl` file as follows:

```diff
actionID = "APIResponse"                                                                 // [!code ++]
name = "API Response Resource"
description = "This resource provides a JSON response through the API."
requires {
    "llmOutput"                                                                    // [!code ++]
}
```

- **`actionID`**: Specifies the resource identifier as `weatherResponseResource`.
- **`requires`**: Declares `llmOutputHelper` as a dependency, ensuring access to its processed output.

### Configuring the `APIResponse` Section

Update the `APIResponse` block to define the structure of the API response:

```diff
APIResponse {
    success = true                               // [!code ++]
    response {
        data {
            "@(llm.response("llmOutput"))"       // [!code ++]
        }
    ...
```

- **`success = true`**: Confirms the successful processing of the request.
- **`data`**: Includes the formatted output from `llmOutputHelper`.

This configuration ensures that the AI agent generates structured, user-friendly JSON responses.

### Updating the Workflow

To ensure proper execution, update the workflow to set the default action to `weatherResponseResource`.

#### Modifying `workflow.pkl`

Open the `workflow.pkl` file and adjust the `targetActionID` field as follows:

```diff
targetActionID = "weatherResponseResource" // [!code ++]
```

By integrating the `response` resource and updating the workflow, the AI agent can deliver polished JSON responses via
the API. This step finalizes the integration and guarantees seamless user interaction.

## Compiling and Running the AI Agent

The final steps involve packaging, building, and running the AI Agent.

### Packaging the AI Agent

To package the AI agent, use the following command:

```bash
kdeps package weather_forecast_ai
```

### Building and Running the AI Agent

Once packaged, build and run the AI agent with:

```bash
kdeps run weather_forecast_ai-1.0.0.kdeps
```

This command initializes the AI agent, making it ready to handle API requests.

## Making an API Request to the AI Agent

You can query the AI agent via an API request as follows:

```bash
curl "http://localhost:3000/api/v1/forecast?q=What+is+the+weather+in+Amsterdam?" -X GET
```

### Expected Response

The AI agent will respond with a JSON object similar to the following:

```json
{
  "errors": [
    {
      "code": 0,
      "message": ""
    }
  ],
  "response": {
    "data": [
      "The current weather in Amsterdam, Netherlands is quite chilly with a temperature of 4.3°C at 14:15 on January
    3rd. The wind speed is moderate at 14.4 km/h from the west-northwest direction. There's no precipitation expected
    for the rest of the day.\n\nLooking ahead to the hourly forecast, it seems that the temperature will fluctuate
    between 1.3°C and 5.9°C throughout the day. Precipitation is expected in short bursts, with a total of 1.20 mm by
    the end of the day. The wind speed will pick up slightly later in the evening.\n\nFor the rest of the week, it's
    expected to be quite cold, with temperatures ranging from 1.3°C to 5.9°C. There might be some light precipitation on
    January 4th, but overall, it should remain dry and chilly throughout the day."
    ]
  },
  "success": true
}
```
