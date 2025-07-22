---
outline: deep
---

# HTTP Client Resource

A `Client` resource allows creating HTTP calls to an API endpoint. It enables passing custom request `Data`, request
`Headers`, and supports HTTP methods such as `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, and `HEAD`.

## Creating a New HTTP Client Resource

To create a new HTTP `Client` resource, you can either generate a new AI agent using the `kdeps new` command or
`scaffold` the resource directly.

Here's how to scaffold a `Client` resource:

```bash
kdeps scaffold [aiagent] client
```

This command will add a `Client` resource into the `aiagent/resources` folder, generating the following folder
structure:

```text
aiagent
└── resources
    └── client.pkl
```

The file includes essential metadata and common configurations, such as [Skip Conditions](../workflow-control/skip.md) and
[Preflight Validations](../workflow-control/validations.md). For more details, refer to the [Common Resource
Configurations](../resources.md#common-resource-configurations) documentation.

## HTTP Client Block

Within the file, you'll find the `HTTPClient` block, which is structured as follows:

```apl
amends "resource.pkl"

ActionID = "clientResource"
Name = "HTTP Client"
Description = "Makes HTTP requests to external APIs"
Category = "api"
Requires { "dataResource" }

Run {
    HTTPClient {
        Method = "GET"
        URL = "https://www.google.com"
        Data {}
        Headers {
            ["X-API-KEY"] = request.header("X-API-KEY")
        }
        TimeoutDuration = 60.s
    }
}
```

Key elements of the `HTTPClient` block include:

- **`Method`**: Specifies the HTTP verb to be used for this API call.
- **`URL`**: Defines the API endpoint.
- **`Data`**: Specifies all the request data to be submitted with this request.
- **`Headers`**: Specifies all the request headers to be submitted with this request.
- **`TimeoutDuration`**: Determines the execution timeout in s (seconds), min (minutes), etc., after which the request will be terminated.

When the resource is executed, you can use client functions like `client.responseBody("id")` to access the response
body. For further details, refer to the [HTTP Client
Functions](../functions-utilities/functions.md#http-client-resource-functions) documentation.
