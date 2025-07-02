---
outline: deep
---

# Working with JSON

Depending on your use case, you can use JSON anywhere in Kdeps. There are primary resources where you can use JSON,
including the API Response Resource (which requires the `APIServerMode` to be enabled), the HTTP client resource, and
the LLM resource.

Kdeps includes JSON helpers to parse and generate JSON documents that you can use in your resources.

## API Server Response

When you create a route in the workflow, the resulting response is always a JSON document.

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

Here, the most important part of this response is the `data` array. In the API Server Response resource, the `data {
... }` block is where you populate your JSON response. This block accepts all native PKL types.

### Creating a JSON Response

There are various ways to create a JSON document for your response. Since the `data` array can accept all types, using
`Mapping` is the most *recommended* way to create a JSON response. Below, we highlight other several options for
creating a JSON document and response.

### Using `Mapping` or `Dynamic` Types for JSON Responses

The native `Mapping` type in PKL is the most recommended approach for creating JSON responses due to its simplicity and
clarity. This method allows you to define key-value pairs and incorporate them directly into the `data` array, making
the process straightforward and efficient.

```apl
local JSONResponse = new Mapping {
    ["currentWeather"] = "@(llm.response("llmWeatherReport"))"
}

...

Data {
    JSONResponse
}

...
```

#### Key Considerations for `Dynamic` and Nested Mappings

When working with `Dynamic` types or nested (including deeply nested) mappings, it's important to understand the
resulting structure. The output will be represented as an OPENAPI schema template, where:

- **Dynamic types** are treated as properties.
- **Nested keys** are translated into elements within the schema.

This distinction ensures compatibility with schema-driven systems while maintaining flexibility in how you define and
structure your JSON responses.

### Creating and Parsing JSON Documents

If you need to preprocess or consume a JSON document prior to a response, Kdeps offers several helper functions in order
to create or parse JSON documents.


#### Using the `JSONRenderDocument` Function

The `document` function `JSONRenderDocument` takes a native PKL object and converts it into JSON. You may want to use
this if you have consumers of the JSON object in other resource, then send this back to the api response resource.

For example, if a `local` variable is declared to contain the JSON structure and content in native PKL types, you can
pass it into the `JSONRenderDocument` function, and it will be converted into a JSON document as a `String`.

```apl
local localWeather = new Mapping {
    ["currentWeather"] = "@(llm.response("llmWeatherReport"))"
}

local JSONResponse = """
@(document.JSONRenderDocument(localWeather))
"""
```

#### Directly Using a `String`

You can create a JSON document response directly from a string. However, unlike using `JSONRenderDocument`, you are
responsible for ensuring that the string is a valid JSON document.

Here’s an example of defining a JSON string:

```apl
local localWeather = """
{
  "currentWeather": "@(llm.response("llmWeatherReport"))"
}
"""
```

This approach is particularly useful when working with static or semi-dynamic JSON strings. However, it requires careful
validation to avoid issues with malformed JSON.

To further process the JSON response, you can:
1. Use `JSONParserMapping` to convert it into a `Mapping` PKL object, which allows you to leverage the `Mapping` API for
   interacting with the JSON content.
2. Use `JSONParser` to convert the string into a `Dynamic` PKL object.

After parsing, you can pass the resulting object into `JSONRenderDocument` to produce a valid JSON document as a
`String`.

Here’s an example of parsing and re-rendering the JSON:

```apl
local parsedJsonMapping = "@(document.JSONParserMapping(localWeather))"

local parsedJsonString = """
@(document.JSONRenderDocument(parsedJsonMapping))
"""
```

This method enables you to validate, manipulate, and reformat the JSON as needed while maintaining control over its
structure and content.

#### Using a Resource Output `file`

Certain resources, such as the LLM Resource, can directly output JSON using the `JSONResponseKeys` configuration. For
other resources, you must ensure that their output is properly formatted as JSON.

To retrieve the execution results, you can use the `file` function, which is available across all resource types. This
function provides the path to the generated output file. The following `file` functions are available for each resource
type:

- `llm.file("id")` - Retrieves the file path for the LLM response output.
- `python.file("id")` - Retrieves the file path for the Python stdout output.
- `client.file("id")` - Retrieves the file path for the HTTP client response body.
- `exec.file("id")` - Retrieves the file path for the Shell execution stdout output.

Once you have obtained the file path, you can use the `read#text` function to read its content.

Here’s an example:

```apl
local llmOutputFilepath = "@(llm.file("llmWeatherReport"))"

local llmOutputJson = """
@(read("\(llmOutputFilepath)")?.text)
"""
```

In this example:
1. The `llm.file` function retrieves the file path of the generated output from the LLM resource.
2. The `read#text` function reads the content of the file, making it available for further processing.

#### LLM Structured Output

When `JSONResponse` is enabled, the LLM can produce a structured output in JSON format. To define the keys for the
structured JSON response, list them in the `JSONResponseKeys` configuration.

You can specify the expected data type for each key by including a type hint in the key name. Examples include:
- `first_name_string` for a string value.
- `famous_quotes_array` for an array.
- `age_integer` for an integer value.

To ensure the output remains valid JSON, consider the following:
1. **Minimize Formatting Issues**: LLM output may occasionally include newlines or special characters that disrupt the
JSON structure. Tailor your prompt to encourage clean and consistent formatting.

2. **Optimize for Structured Output**: Add a hint to your prompt such as, *"Use plain text with no line breaks."* to
   improve the quality and reliability of the structured JSON response.


By combining type hints and prompt optimization, you can ensure the LLM produces well-structured and valid JSON outputs.

## Parsing JSON Strings

To parse a JSON string, use the `JSONParser` function. This converts the string into a `Dynamic` PKL
object. Alternatively, you can use `JSONParserMapping` to output a `Mapping` PKL object.

Here’s an example:

```apl
local localWeatherJson = """
{
  "currentWeather": "@(llm.response("llmWeatherReport"))"
}
"""
local localWeather = "@(JSONParser(localWeatherJson))"
local currentWeather = "@(localWeather?.currentWeather)"
```

In this example:

- `JSONParser` processes the `localWeatherJson` string and converts it into a `Dynamic` PKL object.
- The `currentWeather` value is then extracted from the parsed object using `localWeather?.currentWeather`.

Similarly, if you use `JSONParserMapping`, the JSON string is converted into a `Mapping` PKL object, allowing you to
leverage the `Mapping` API for more structured and type-safe access to the data.

Here’s an example:

```apl
local localWeatherJson = """
{
  "currentWeather": "@(llm.response("llmWeatherReport"))"
}
"""
local localWeather = "@(JSONParserMapping(localWeatherJson))"
local currentWeather = "@(localWeather.getOrNull("currentWeather"))"
```

In this example:
- `JSONParserMapping` transforms the `localWeatherJson` string into a `Mapping` PKL object.
- The `Mapping` API's `getOrNull` method is used to access the `currentWeather` value.
