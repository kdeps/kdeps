---
outline: deep
---

# Creating Structured LLM Response APIs

In this tutorial, we’ll explore how to design structured responses for your LLMs when building with Kdeps AI
Agents. Structured JSON output is essential for creating reliable APIs, as it ensures data consistency, simplifies
parsing, and enhances integration with downstream systems.

We’ll enable the [`apiServeMode`](/getting-started/configuration/workflow#api-server-settings) setting in this example,
but the same approach can also be applied in [Lambda Mode](/getting-started/configuration/workflow#lambda-mode).

## Getting Started

Let’s begin by setting up our project. This tutorial will guide you through scaffolding four essential `.pkl` files:

- **`workflow.pkl`**: Configures the AI Agent. See details in [Workflow](/getting-started/configuration/workflow).
- **`resources/llm.pkl`**: Defines the LLM model responsible for generating structured JSON output. Learn more in [LLM
  Resource](/getting-started/resources/functions#llm-resource-functions).
- **`resources/response.pkl`**: Handles structured JSON output responses. Check out [Response
  Resource](/getting-started/resources/response).
- **`resources/client.pkl`**: Creates an HTTP Client to call an API.

By the end of this tutorial, you’ll have a foundational understanding of how to structure LLM responses for API-based
applications.

## Setting Up Our AI Agent: `weather_forecast_ai`

To begin, we will scaffold the resources for our project. We'll name our AI agent `weather_forecast_ai` and use the following command to generate the necessary files all at once:

```bash
kdeps scaffold weather_forecast_ai workflow llm response client
```

This command will create the following directory structure:

```plaintext
weather_forecast_ai
├── data
├── resources
│   ├── client.pkl
│   ├── llm.pkl
│   └── response.pkl
└── workflow.pkl
```

### Updating the `workflow.pkl` File

Next, let's modify the `workflow.pkl` file by adding a new route to the `apiServer` block. Here's the updated configuration:

```diff
apiServer {
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

First, let's rename the `resources/llm.pkl` file to `resources/llm_input_helper.pkl`:

```bash
mv weather_forecast_ai/resources/llm.pkl \
   weather_forecast_ai/resources/llm_input_helper.pkl
```

Next, create a duplicate of this file, naming it `resources/llm_output_helper.pkl`. This will handle output preparation:

```bash
cp weather_forecast_ai/resources/llm_input_helper.pkl \
   weather_forecast_ai/resources/llm_output_helper.pkl
```

### Choosing the Weather API

We'll use [Open Mateo](https://open-mateo.com) for this project, as it offers free API access. If you prefer a different weather API, you'll need to obtain an API key, define it as `args` in the `workflow.pkl` file, and add the key to your `.env` file.

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
https://localhost:3000/api/v1/forecast?q=What+is+the+weather+in+Amsterdam?
```

### Updating `llm_input_helper.pkl`

Open the `resources/llm_input_helper.pkl` file and update the resource details as follows:

```diff
id = "llmInputHelper"                                            // [!code ++]
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
@(request.param("q"))?                                    // [!code ++]
"""
    jsonResponse = true                                   // [!code ++]
    jsonResponseKeys {
        "longitude_str"                                   // [!code ++]
        "latitude_str"                                    // [!code ++]
        "timezone_str"                                    // [!code ++]
    }
...
}
```

Key points:

- **`@(request.param("q"))`**: This function extracts the query parameter with the ID `q`.
- **Appended `_str` to Response Keys**: Adding `_str` enforces a typed structure for the LLM output.

By following these steps, you'll have a structured and type-safe LLM for interacting with the Weather API.

Here’s an improved and rephrased version of your text:

---

## Creating an HTTP Client for the Weather API

In this step, we will build the HTTP client responsible for interacting with the Weather API. Using the structured output (`longitude_str`, `latitude_str`, and `timezone_str`) from our previous LLM call, we’ll extract these values with PKL's internal JSON library.

### Editing `resources/client.pkl`

First, update the `resources/client.pkl` file as follows:

```diff
id = "httpWeatherClientResource"                                                   // [!code ++]
name = "HTTP Client for the Weather API"
description = "This resource enables API requests to the Weather API."
requires {
    "llmInputHelper"                                                               // [!code ++]
}
```

By defining `llmInputHelper` as a dependency, this resource can access its outputs.

### Adding a JSON Parser

In the same file, add a JSON parser to extract the necessary values:

```diff
import "pkl:json"                                                // [!code ++]
local jsonParser = new json.Parser {}                            // [!code ++]
local jsonData = """                                             // [!code ++]
@(llm.response("llmInputHelper"))                                // [!code ++]
"""                                                              // [!code ++]
local latitude = "@(jsonParser.parse(jsonData).latitude_str)"    // [!code ++]
local longitude = "@(jsonParser.parse(jsonData).longitude_str)"  // [!code ++]
local timezone = "@(jsonParser.parse(jsonData).timezone_str)"    // [!code ++]
```

Here, the `jsonParser` object processes the structured output from `llmInputHelper`. The fields `latitude_str`, `longitude_str`, and `timezone_str` are parsed and stored in variables.

### Defining the HTTP Client

Now, define the `httpClient` block to handle API calls:

```diff
local api_endpoint = "https://api.open-meteo.com/v1/forecast"                          // [!code ++]
local api_weather_params = """                                                         // [!code ++]
&current_weather=true                                                                  // [!code ++]
&hourly=temperature_2m,precipitation,wind_speed_10m                                    // [!code ++]
&daily=temperature_2m_max,temperature_2m_min,precipitation_sum                         // [!code ++]
"""                                                                                    // [!code ++]
local api_params = "?latitude=\(latitude)&longitude=\(longitude)&timezone=\(timezone)" // [!code ++]
                                                                                       // [!code ++]
httpClient {
    method = "GET"                                                                     // [!code ++]
    url = "\(api_endpoint)\(api_params)\(api_weather_params)"                          // [!code ++]
...
```

- **`api_endpoint`**: Defines the base URL for the Weather API.
- **`api_weather_params`**: Specifies the required data (current weather, hourly details, and daily summaries).
- **`api_params`**: Dynamically constructs query parameters using `latitude`, `longitude`, and `timezone`.
- **`httpClient` block**: Configures the HTTP GET request by combining the endpoint, parameters, and weather details.

By completing these steps, you’ll have an HTTP client ready to make API requests to retrieve detailed weather
information based on structured input.

## Constructing the Output Using LLM

With the JSON response from the Weather API in hand, let's format it into a user-friendly output using the LLM.

### Updating `llm_output_helper.pkl`

Open the `resources/llm_output_helper.pkl` file and update the resource details as follows:

```diff
id = "llmOutputHelper"                                              // [!code ++]
name = "AI Helper for Output"
description = "A resource to generate a polished output using LLM."
requires {
    "httpWeatherClientResource"                                     // [!code ++]
}
```

Here, we define a dependency on `httpWeatherClientResource`, ensuring the LLM has access to the Weather API response.

---

### Adding the Output Formatting Logic

Next, configure the output construction logic:

```diff
chat {
    model = "llama3.1"                                                      // [!code ++]
    prompt = """
As if you're a weather reporter, present this response in an engaging way:  // [!code ++]
@(client.responseBody("httpWeatherClientResource"))                         // [!code ++]
"""
    jsonResponse = false                                                    // [!code ++]
...
```

- **`model`**: Specifies the LLM version (`llama3.1`).
- **`prompt`**: Instructs the model to generate an output resembling a weather reporter's announcement. The raw API
response is passed via `@(client.responseBody("httpWeatherClientResource"))`.
- **`jsonResponse = false`**: Indicates that the output should be plain text, formatted for readability, rather than
structured JSON.

This setup ensures that the output from the Weather API is transformed into an engaging and easy-to-understand
narrative, making it more relatable for end users.

## Finalizing the AI Agent API

To complete the AI agent, we’ll incorporate a `response` resource that enables the API to deliver structured JSON responses.

### Updating `resources/response.pkl`

Edit the `resources/response.pkl` file as follows:

```diff
id = "weatherResponseResource"                                                     // [!code ++]
name = "API Response Resource"                                                     // [!code ++]
description = "This resource provides a JSON response through the API."            // [!code ++]
requires {                                                                         // [!code ++]
    "llmOutputHelper"                                                              // [!code ++]
}
```

- **`id`**: Specifies the resource identifier as `weatherResponseResource`.
- **`requires`**: Declares `llmOutputHelper` as a dependency, ensuring access to its processed output.

### Configuring the `apiResponse` Section

Update the `apiResponse` block to define the structure of the API response:

```diff
apiResponse {
    success = true                               // [!code ++]
    response {
        data {
            "@(llm.response("llmOutputHelper"))" // [!code ++]
        }
    ...
```

- **`success = true`**: Confirms the successful processing of the request.
- **`data`**: Includes the formatted output from `llmOutputHelper`.

This configuration ensures that the AI agent generates structured, user-friendly JSON responses.

### Updating the Workflow

To ensure proper execution, update the workflow to set the default action to `weatherResponseResource`.

#### Modifying `workflow.pkl`

Open the `workflow.pkl` file and adjust the `action` field as follows:

```diff
action = "weatherResponseResource" // [!code ++]
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
  ...
}
```

This response will include the weather forecast details in a structured and user-friendly format.
