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

```apl
local bearerToken = """
@(read?("file:/tmp/bearer.txt")?.text)
"""

SkipCondition {
    bearerToken.length != 0 // If the bearerToken file contains data,
                            // authentication is unnecessary.
}
```

### Example 2: Targeting a Specific API Endpoint

Another common use case is handling multiple API endpoints and selectively executing resources for specific endpoints.

Here, the resource only runs if the `requestPath` matches the specified `allowedPath`.

```apl
local allowedPath = "/api/v1/items"
local requestPath = "@(request.path())"

SkipCondition {
    requestPath != allowedPath // Skip execution for paths other than the allowedPath.
}
```

By defining `SkipCondition` rules tailored to your requirements, you can optimize resource execution and ensure
efficient handling of diverse scenarios.

## Using Skip Condition Helpers

Skip Condition Helpers provide utility functions to streamline the process of defining skip rules. These helpers
simplify common checks and improve code readability. Below is a growing list of available helpers:

| **Function**                          | **Description**                                                                            |
|---------------------------------------|--------------------------------------------------------------------------------------------|
| `skip.ifFileExists("string")`         | Returns `true` if the specified file exists; `false` otherwise.                            |
| `skip.ifFolderExists("string")`       | Returns `true` if the specified folder exists; `false` otherwise.                          |
| `skip.ifFileIsEmpty("string")`        | Returns `true` if the specified file is empty; `false` otherwise.                          |

These helpers enable you to create concise and expressive skip conditions, improving maintainability and clarity in your
resource definitions.
