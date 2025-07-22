---
outline: deep
---

# How to Build a Weather API with Kdeps

This tutorial demonstrates how to build a Weather API using Kdeps. The API will accept location queries, extract coordinates using an LLM, fetch weather data from a third-party API, and return formatted responses.

## Overview

The Weather API will:
1. Accept location queries (e.g., "What's the weather in Amsterdam?")
2. Use an LLM to extract coordinates and timezone from the query
3. Fetch weather data from the Open-Meteo API
4. Format the response using another LLM
5. Return a structured JSON response

## Prerequisites

- Kdeps installed and configured
- Basic understanding of PKL syntax
- Access to the internet for API calls

## Project Structure

```
weather_forecast_ai/
├── workflow.pkl
├── resources/
│   ├── llm_input.pkl
│   ├── exec.pkl
│   ├── client.pkl
│   ├── llm_output.pkl
│   └── response.pkl
└── .kdeps.pkl
```

## Step 1: Creating the AI Agent

First, create a new AI agent:

```bash
kdeps new weather_forecast_ai
cd weather_forecast_ai
```

## Step 2: Setting Up Resources

### Creating the LLM Input Resource

Create the first resource to handle input parsing:

```bash
kdeps scaffold weather_forecast_ai llm
```

This creates `resources/llm_input.pkl`.

### Renaming and Duplicating Files

Rename the generated file to `resources/llm_input.pkl` to better reflect its purpose.

Next, create a duplicate of this file, naming it `resources/llm_output.pkl`. This will handle output preparation:

```bash
cp resources/llm_input.pkl resources/llm_output.pkl
```

### Creating Additional Resources

Create the remaining resources:

```bash
kdeps scaffold weather_forecast_ai exec
kdeps scaffold weather_forecast_ai client
kdeps scaffold weather_forecast_ai response
```

## Step 3: Configuring the LLM Input Resource

The LLM input resource will parse location queries and extract structured data.

### Updating Resource Details

Open the `resources/llm_input.pkl` file and update the resource details as follows:

```apl
amends "resource.pkl"

ActionID = "llmInput"
Name = "AI Helper for Input"
Description = "An AI helper to parse input into structured data"
Category = "ai"
Requires { "dataResource" }

Run {
    Chat {
        Model = "llama3.1"
        Prompt = """
Extract the longitude, latitude, and timezone
from this text. An example of timezone is Asia/Manila.
\(request.params("q"))?
"""
        JSONResponse = true
        JSONResponseKeys {
            "longitude_str"
            "latitude_str"
            "timezone_str"
        }
        TimeoutDuration = 60.s
    }
}
```

### Adding Model, Prompt, and Response Keys

Next, define the model, prompt, and structured response keys:

```diff
Chat {
    Model = "llama3.1"
    Prompt = """
Extract the longitude, latitude, and timezone
from this text. An example of timezone is Asia/Manila.
\(request.params("q"))?
"""
    JSONResponse = true
    JSONResponseKeys {
        "longitude_str"
        "latitude_str"
        "timezone_str"
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

```apl
amends "resource.pkl"

ActionID = "execResource"
Name = "Store LLM JSON response to a file"
Description = "This resource will store the LLM JSON response to a file for processing later"
Category = "system"
Requires { "llmInput" }

Run {
    Exec {
        Command = """
        rm -rf /tmp/llm_input.json
        echo $LLM_INPUT > /tmp/llm_input.json
        """
        Env {
            ["LLM_INPUT"] = llm.response("llmInput")
        }
        TimeoutDuration = 60.s
    }
}
```

By defining `llmInput` as a dependency, this resource can access its outputs.

### Creating the JSON file

Let's say `/tmp/llm_input.json` is the name of the file, we need to create this file from the output of the `llmInput`.
We also ensure that this file is recreated by deleting it first, so that we are sure that it's a fresh file, not a
previously generated file that we might have reused.

```diff
Exec {
    Command = """
    rm -rf /tmp/llm_input.json
    echo $LLM_INPUT > /tmp/llm_input.json
    """
    Env {
        ["LLM_INPUT"] = "@(llm.response("llmInput"))"
    }
```

## Creating an HTTP Client for the Weather API

In this step, we will build the HTTP client responsible for interacting with the Weather API. Using the structured
output (`longitude_str`, `latitude_str`, and `timezone_str`) from our previous LLM call, we'll extract these values the
built-in JSON parser.

### Editing `resources/client.pkl`

First, update the `resources/client.pkl` file as follows:

```apl
amends "resource.pkl"

ActionID = "HTTPClient"
Name = "HTTP Client for the Weather API"
Description = "This resource enables API requests to the Weather API."
Category = "api"
Requires { "execResource" }

Run {
    local JSONData = """
    \(read?("file:/tmp/llm_input.json")?.text)
    """

    HTTPClient {
        Method = "GET"
        URL = "https://api.open-meteo.com/v1/forecast"
        Data {}
        Params {
            ["latitude" ] = JSONParser.parse(JSONData)?.latitude_str
            ["longitude"] = JSONParser.parse(JSONData)?.longitude_str
            ["timezone "] = JSONParser.parse(JSONData)?.timezone_str
            ["current_weather"] = "true"
            ["forecast_days"] = "1"
            ["hourly"] = "temperature_2m,precipitation,wind_speed_10m"
            ["daily"] = "temperature_2m_max,temperature_2m_min,precipitation_sum"
        }
        TimeoutDuration = 60.s
    }
}
```

### Defining the HTTP Client

Now, define the `HTTPClient` block to handle API calls:

```diff
HTTPClient {
    Method = "GET"
    URL = "https://api.open-meteo.com/v1/forecast"
    Params {
        ["current_weather"] = "true"
        ["forecast_days"] = "1"
        ["hourly"] = "temperature_2m,precipitation,wind_speed_10m"
        ["daily"] = "temperature_2m_max,temperature_2m_min,precipitation_sum"
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
local JSONData = """
@(read?("file:/tmp/llm_input.json")?.text)
"""

HTTPClient {
    Method = "GET"
    URL = "https://api.open-meteo.com/v1/forecast"
    Data {}
    Params {
        ["latitude" ] = "@(JSONParser.parse(JSONData)?.latitude_str)"
        ["longitude"] = "@(JSONParser.parse(JSONData)?.longitude_str)"
        ["timezone "] = "@(JSONParser.parse(JSONData)?.timezone_str)"
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
ActionID = "llmOutput"
Name = "AI Helper for Output"
Description = "A resource to generate a polished output using LLM."
Requires {
    "HTTPClient"
}
```

### Adding the Output Formatting Logic

Next, configure the output construction logic:

```diff
Chat {
    Model = "llama3.1"
    Prompt = """
As if you're a weather reporter, present this response in an engaging way:
\(client.responseBody("HTTPClient"))
"""
    JSONResponse = false
...
```

- **`Model`**: Specifies the LLM version (`llama3.1`).
- **`Prompt`**: Instructs the model to generate an output resembling a weather reporter's announcement. The raw API
response is passed via `\(client.responseBody("HTTPClient"))`.
- **`JSONResponse = false`**: Indicates that the output should be plain text, formatted for readability, rather than
structured JSON.

This setup ensures that the output from the Weather API is transformed into an engaging and easy-to-understand
narrative, making it more relatable for end users.

## Finalizing the AI Agent API

To complete the AI agent, we'll incorporate a `response` resource that enables the API to deliver structured JSON responses.

### Updating `resources/response.pkl`

Edit the `resources/response.pkl` file as follows:

```diff
ActionID = "APIResponse"
Name = "API Response Resource"
Description = "This resource provides a JSON response through the API."
Requires {
    "llmOutput"
}
```

- **`ActionID`**: Specifies the resource identifier as `weatherResponseResource`.
- **`Requires`**: Declares `llmOutputHelper` as a dependency, ensuring access to its processed output.

### Configuring the `APIResponse` Section

Update the `APIResponse` block to define the structure of the API response:

```diff
APIResponse {
    Success = true
    Response {
        Data {
            llm.response("llmOutput")
        }
    }
}
```

- **`Success = true`**: Confirms the successful processing of the request.
- **`Data`**: Includes the formatted output from `llmOutputHelper`.

This configuration ensures that the AI agent generates structured, user-friendly JSON responses.

### Updating the Workflow

To ensure proper execution, update the workflow to set the default action to `weatherResponseResource`.

#### Modifying `workflow.pkl`

Open the `workflow.pkl` file and adjust the `TargetActionID` field as follows:

```diff
TargetActionID = "weatherResponseResource"
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
