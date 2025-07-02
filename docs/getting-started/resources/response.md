# API Response Resource

The `Response` resource is designed to initialize an API response in JSON format by pre-filling the `Data` array with
values generated from previously executed resources.

Using this resource alongside [Skip Conditions](../resources/skip) and [Preflight
Validations](../resources/validations), you can define rules to output either a successful response or a custom `Error`.

You can define multiple API routes in the `workflow.pkl` file. For more information, refer to the
[Workflow](../configuration/workflow) documentation.

## Creating a New API Response Resource

To create a new `Response` resource, you can either generate a new AI agent using the `kdeps new` command or scaffold
the resource directly.

Here's how to scaffold a `Response` resource:

```bash
kdeps scaffold [aiagent] response
```

This command will add a `Response` resource to the `aiagent/resources` folder, creating the following folder structure:

```bash
aiagent
└── resources
    └── response.pkl
```

The generated file includes essential metadata and common configurations, such as [Skip Conditions](../resources/skip)
and [Preflight Validations](../resources/validations). For more details, refer to the [Common Resource
Configurations](../resources/resources#common-resource-configurations) documentation.

## API Response Block

The file contains the `APIResponse` block, structured as follows:

```apl
APIResponse {
    Success = true
    Meta {
        Headers {
            // ["X-Frame-Options"] = "DENY"
            // ["Content-Security-Policy"] = "default-src 'self'; connect-src *; font-src *; script-src-elem * 'unsafe-inline'; img-src * data:; style-src * 'unsafe-inline';"
            // ["X-XSS-Protection"] = "1; mode=block"
            // ["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains; preload"
            // ["Referrer-Policy"] = "strict-origin"
            // ["X-Content-Type-Options"] = "nosniff"
            // ["Permissions-Policy"] = "geolocation=(),midi=(),sync-xhr=(),microphone=(),camera=(),magnetometer=(),gyroscope=(),fullscreen=(self),payment=()"
        }
        Properties {
            // ["X-Custom-Properties"] = "value"
        }
    }
    Response {
        Data {
            "@(llm.response("llmResource"))"
            // "@(python.stdout("pythonResource"))"
            // "@(exec.stdout("shellResource"))"
            // "@(client.responseBody("httpResource"))"
        }
    }
    Errors {
        new {
            Code = 0
            Message = ""
        }
    }
}
```

Key Elements of the `APIResponse` Block:

- **`Success`**: Indicates whether the response signifies a successful operation.
- **`Meta`**: Meta block includes the `custom response headers`, `custom response properties`, and `requestID`.
- **`Response`**: Populates the response `Data` with outputs from resources such as `llm`, `python`, `exec`, or
  `client`.
- **`Errors`**: Defines custom error codes and messages to handle various error cases. Multiple errors can be defined
  and returned.
- **`TimeoutDuration`**: Sets the timeout duration in seconds, after which the execution will be terminated.
