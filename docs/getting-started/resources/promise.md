---
outline: deep
---

# Promise Operator

Kdeps uses the `"@()"` post-execution template convention to defer the execution of resource functions to a later stage.

Since Kdeps relies on a number of [Apple PKL](https://pkl-lang.org) functions, any function that depends on values
generated during resource execution must be wrapped in this template convention.

> *Note:* The `"@()"` operator must always be enclosed in double quotes. In Apple PKL, it is treated as a string, which is
> later post-processed by Kdeps.

Without the promise operator, such functions would execute prematurely, causing an exception.

The promise operator is commonly used in [Resources](../resources/resources.md). Below are examples of its applications:

## Skip Condition

Each resource includes a `skipCondition` block that, when evaluated as `true`, skips the resource's execution.

In this example, the `@(request.path())` expression is wrapped with the promise operator to ensure the value is deferred:

```apl
local allowedPath = "/api/v1/items"
local requestPath = "@(request.path())"

SkipCondition {
    requestPath != allowedPath
}
```

## Preflight Validations

In this scenario, a resource requires the uploaded file attachment to be of specific types—PDF, PNG, or JPEG.

The promise operator is used to evaluate the MIME type of the uploaded file, as shown below:

```apl
local fileType = "@(request.filetypes()[0])"

PreflightCheck {
    Validations {
        fileType == "application/pdf" || fileType == "image/png" || fileType == "image/jpeg"
    }
}
```

## Resource Functions

All resource functions must use the promise operator. For further information, see the [Functions](../resources/functions) documentation.

## API Request Functions

Similarly, API request functions require the promise operator. For additional details, refer to the [API Request Functions](../resources/functions#api-request-functions) documentation.
