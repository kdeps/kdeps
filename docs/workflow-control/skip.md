---
outline: deep
---

# Skip Conditions

Skip conditions provide a way to define criteria for bypassing the execution of a resource.

They are particularly useful in scenarios where executing a resource would be redundant or unnecessary based on specific
circumstances.

The logic for skip conditions is based on an `OR` operation, meaning the execution will be skipped if **any** of the
defined conditions are met.

Additionally, it can accept either a string value of `"true"` (case-insensitive) or a Boolean `true`. Any other input
will result in a Boolean `false`.

## Defining a `SkipCondition`

To create a `SkipCondition`, assign a function to a local variable that evaluates the condition. Here's an example:

### Example 1: Skipping Authentication if a Bearer Token Exists

In this scenario, we check for the presence of a bearer token. If it exists, the authentication step is skipped.

```pkl
local bearerToken = read?("file:/tmp/bearer.txt")?.text

SkipCondition {
    skip.ifNotEmpty(bearerToken) // If the bearerToken file contains data,
                                 // authentication is unnecessary.
}
```

### Example 2: Targeting a Specific API Endpoint

Another common use case is handling multiple API endpoints and selectively executing resources for specific endpoints.

Here, the resource only runs if the `requestPath` matches the specified `allowedPath`.

```pkl
local allowedPath = "/api/v1/items"
local requestPath = request.path()

SkipCondition {
    !skip.ifEquals(requestPath, allowedPath) // Skip execution for paths other than the allowedPath.
}
```

By defining `SkipCondition` rules tailored to your requirements, you can optimize resource execution and ensure
efficient handling of diverse scenarios.

## Using Skip Condition Helpers

Skip Condition Helpers provide utility functions to streamline the process of defining skip rules. These helpers
simplify common checks and improve code readability. Below is a growing list of available helpers:

| **Function**                                      | **Description**                                                                                     |
|---------------------------------------------------|-----------------------------------------------------------------------------------------------------|
| **File & Folder Operations**                     |                                                                                                     |
| `skip.ifFileExists("path")`                      | Returns `true` if the specified file exists; `false` otherwise.                                    |
| `skip.ifFolderExists("path")`                    | Returns `true` if the specified folder exists and contains files; `false` otherwise.              |
| `skip.ifFileIsEmpty("path")`                     | Returns `true` if the specified file exists and is empty; `false` otherwise.                      |
| `skip.ifFileNotEmpty("path")`                    | Returns `true` if the specified file exists and contains content; `false` otherwise.              |
| `skip.ifFileContains("path", "text")`            | Returns `true` if the file exists and contains the specified text; `false` otherwise.             |
| **String Comparisons**                           |                                                                                                     |
| `skip.ifEquals("value", "expected")`             | Returns `true` if the values are equal (case-sensitive); `false` otherwise.                       |
| `skip.ifEqualsIgnoreCase("value", "expected")`   | Returns `true` if the values are equal (case-insensitive); `false` otherwise.                     |
| `skip.ifEmpty("value")`                          | Returns `true` if the value is null, empty, or whitespace only; `false` otherwise.                |
| `skip.ifNotEmpty("value")`                       | Returns `true` if the value is not null and not empty; `false` otherwise.                         |
| `skip.ifStartsWith("value", "prefix")`           | Returns `true` if the value starts with the specified prefix; `false` otherwise.                  |
| `skip.ifEndsWith("value", "suffix")`             | Returns `true` if the value ends with the specified suffix; `false` otherwise.                    |
| `skip.ifContains("value", "text")`               | Returns `true` if the value contains the specified text; `false` otherwise.                       |
| **Numeric Comparisons**                          |                                                                                                     |
| `skip.ifGreaterThan("value", "threshold")`       | Returns `true` if the numeric value is greater than the threshold; `false` otherwise.             |
| `skip.ifLessThan("value", "threshold")`          | Returns `true` if the numeric value is less than the threshold; `false` otherwise.                |
| **Value Comparisons**                            |                                                                                                     |
| `skip.ifValueMatches("value", "pattern")`        | Returns `true` if the value matches the pattern; `false` otherwise.                               |
| `skip.ifValueIsMethod("value", "method")`        | Returns `true` if the value matches the HTTP method (case-insensitive); `false` otherwise.       |
| `skip.ifValuesEqual("value", "expected")`        | Returns `true` if two values are equal; `false` otherwise.                                        |
| `skip.ifValueExists("value")`                    | Returns `true` if the value exists and is not empty; `false` otherwise.                           |

These helpers enable you to create concise and expressive skip conditions, improving maintainability and clarity in your
resource definitions.

## Practical Examples

### Example 3: Skip Based on File Content
Skip execution if a configuration file contains a specific setting:

```pkl
SkipCondition {
    skip.ifFileContains("/tmp/config.json", "\"enabled\": false")
}
```

### Example 4: Skip Based on Request Method
Only execute for POST requests:

```pkl
local requestMethod = request.method()
SkipCondition {
    !skip.ifValueIsMethod(requestMethod, "POST")  // Skip if NOT a POST request
}
```

### Example 5: Skip Based on Authentication Header
Skip if no authorization header is present:

```pkl
local authHeader = request.headers("Authorization")
SkipCondition {
    !skip.ifValueExists(authHeader)  // Skip if no auth header
}
```

### Example 6: Skip Based on Data Length
Skip if the request data is too short:

```pkl
local requestData = request.data()
local dataLength = requestData.length.toString()
SkipCondition {
    skip.ifLessThan(dataLength, "10")  // Skip if data < 10 characters
}
```

### Example 7: Skip Based on Environment Variables
Skip in development environment:

```pkl
local environment = read?("/tmp/environment.txt")?.text
SkipCondition {
    skip.ifEqualsIgnoreCase(environment, "development")
}
```

### Example 8: Skip Based on API Version
Skip for older API versions:

```pkl
local requestPath = request.path()
SkipCondition {
    skip.ifStartsWith(requestPath, "/api/v1/")  // Skip v1 API calls
}
```
