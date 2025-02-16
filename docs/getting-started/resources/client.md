---
outline: deep
---

# HTTP Client Resource

A `client` resource allows creating HTTP calls to an API endpoint. It enables passing custom request `data`, request
`headers`, and supports HTTP methods such as `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, and `HEAD`.

## Creating a New HTTP Client Resource

To create a new HTTP `client` resource, you can either generate a new AI agent using the `kdeps new` command or
`scaffold` the resource directly.

Here’s how to scaffold a `client` resource:

```bash
kdeps scaffold [aiagent] client
```

This command will add a `client` resource into the `aiagent/resources` folder, generating the following folder
structure:

```text
aiagent
└── resources
    └── client.pkl
```

The file includes essential metadata and common configurations, such as [Skip Conditions](../resources/skip) and
[Preflight Validations](../resources/validations). For more details, refer to the [Common Resource
Configurations](../resources/resources#common-resource-configurations) documentation.

## HTTP Client Block

Within the file, you’ll find the `HTTPClient` block, which is structured as follows:

```apl
HTTPClient {
    method = "GET"
    url = "https://www.google.com"
    data {}
    headers {
        ["X-API-KEY"] = "@(request.header("X-API-KEY"))"
    }
    timeoutDuration = 60.s
}
```

Key elements of the `HTTPClient` block include:

- **`method`**: Specifies the HTTP verb to be used for this API call.
- **`url`**: Defines the API endpoint.
- **`data`**: Specifies all the request data to be submitted with this request.
- **`headers`**: Specifies all the request headers to be submitted with this request.
- **`timeoutDuration`**: Determines the exectuion timeout in s (seconds), min (minutes), etc., after which the request will be terminated.

When the resource is executed, you can use client functions like `client.responseBody("id")` to access the response
body. For further details, refer to the [HTTP Client
Functions](../resources/functions.md#http-client-resource-functions) documentation.
