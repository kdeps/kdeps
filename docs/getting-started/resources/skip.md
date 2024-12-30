---
outline: deep
---

# Skip Conditions

Skip conditions allow you to specify criteria for bypassing the execution of a resource.

These are particularly helpful in situations where executing a resource is unnecessary or redundant under certain
conditions.

### Defining a `skipCondition`

To create a `skipCondition`, assign a function to a local variable that evaluates the condition. Here's an example:

#### Example 1: Skipping Authentication if a Bearer Token Exists

In this scenario, we check for the presence of a bearer token. If it exists, the authentication step is skipped.

```apl
local bearerToken = """
@(read?("file:/tmp/bearer.txt")?.text)
"""

skipCondition {
    bearerToken.length != 0 // If the bearerToken file contains data,
                            // authentication is unnecessary.
}
```

#### Example 2: Targeting a Specific API Endpoint

Another common use case is handling multiple API endpoints and selectively executing resources for specific endpoints.

Here, the resource only runs if the `requestPath` matches the specified `allowedPath`.

```apl
local allowedPath = "/api/v1/items"
local requestPath = "@(request.path())"

skipCondition {
    requestPath != allowedPath // Skip execution for paths other than the allowedPath.
}
```

By defining `skipCondition` rules tailored to your requirements, you can optimize resource execution and ensure
efficient handling of diverse scenarios.
