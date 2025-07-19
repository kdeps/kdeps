---
outline: deep
---

# API Request Validations

API request validations are a critical mechanism for ensuring that incoming API requests meet specific criteria before a resource action is executed. These validations verify the request's HTTP method, URL path, headers, and query parameters against predefined restrictions.

These checks safeguard system integrity, enforce security policies, and streamline workflows by skipping actions that do not comply with the specified requirements. They are particularly relevant when operating in API server mode (`APIServerMode` enabled).

## Why API Request Validations Matter

- **Enforce Request Compliance:** Validations ensure that only requests with permitted methods, paths, headers, and parameters are processed, reducing the risk of unauthorized or malformed requests.
- **Early Action Skipping:** By validating requests before execution, non-compliant actions are skipped early, saving system resources and preventing unintended behavior.
- **Improved Debugging:** When an action is skipped due to a validation failure, detailed log messages help diagnose the issue, such as identifying an invalid HTTP method or path.

## Defining API Request Validations

API request validations are defined in the `Run` block of a resource configuration and are enforced only when `APIServerMode` is enabled. They consist of four key fields:

- `RestrictToHTTPMethods`: Specifies the HTTP methods (e.g., `GET`, `POST`) required for the request.
- `RestrictToRoutes`: Specifies the URL paths (e.g., `/api/v1/whois`) required for the request.
- `AllowedHeaders`: Specifies the HTTP headers permitted in the request.
- `AllowedParams`: Specifies the query parameters permitted in the request.

If any of these fields are empty, all corresponding values are permitted (e.g., an empty `RestrictToHTTPMethods` allows all HTTP methods). If a validation fails, the action is skipped, and no further processing (e.g., `Exec`, `Python`, `Chat`, or `HTTPClient` steps) occurs for that action.

Here's an example of how to configure API request validations:

```apl
Run {
    // RestrictToHTTPMethods specifies the HTTP methods required for the request.
    // If none are specified, all HTTP methods are permitted. This restriction is only
    // in effect when APIServerMode is enabled. If the request method is not in this list,
    // the action will be skipped.
    RestrictToHTTPMethods {
        "GET"
    }

    // RestrictToRoutes specifies the URL paths required for the request.
    // If none are specified, all routes are permitted. This restriction is only
    // in effect when APIServerMode is enabled. If the request path is not in this list,
    // the action will be skipped.
    RestrictToRoutes {
        "/api/v1/whois"
    }

    // AllowedHeaders specifies the permitted HTTP headers for the request.
    // If none are specified, all headers are allowed. This restriction is only
    // in effect when APIServerMode is enabled. If a header used in the resource is not
    // in this list, the action will be skipped.
    AllowedHeaders {
        "Content-Type"
        // "X-API-KEY"
    }

    // AllowedParams specifies the permitted query parameters for the request.
    // If none are specified, all parameters are allowed. This restriction is only
    // in effect when APIServerMode is enabled. If a parameter used in the resource is
    // not in this list, the action will be skipped.
    AllowedParams {
        "user_id"
        "session_id"
    }
}
```

### Validation Details

- **RestrictToHTTPMethods**:
  - Validates the request's HTTP method (e.g., `GET`, `POST`) against the specified list.
  - Example: If set to `["GET"]`, a `POST` request will cause the action to be skipped.
  - Case-insensitive matching is used (e.g., `get` matches `GET`).

- **RestrictToRoutes**:
  - Validates the request's URL path (e.g., `/api/v1/whois`) against the specified list.
  - Example: If set to `["/api/v1/whois"]`, a request to `/api/v1/users` will cause the action to be skipped.
  - Exact path matching is used; patterns or wildcards are not currently supported.

- **AllowedHeaders**:
  - Validates headers used in `request.header("header_id")` calls within the resource file against the specified list.
  - Example: If set to `["Content-Type"]`, a `request.header("Authorization")` call will cause the action to be skipped.
  - Case-insensitive matching is used.

- **AllowedParams**:
  - Validates query parameters used in `request.params("param_id")` calls within the resource file against the specified list.
  - Example: If set to `["user_id"]`, a `request.params("token")` call will cause the action to be skipped.
  - Case-insensitive matching is used.

### Behavior in APIServerMode

- **Enabled (`APIServerMode = true`)**:
  - All validations are enforced.
  - If any validation fails, the action is skipped, and a log message is recorded (e.g., "Skipping action due to method validation failure").
  - The workflow continues processing the next resource in the dependency stack.

- **Disabled (`APIServerMode = false`)**:
  - Validations are bypassed, and all HTTP methods, routes, headers, and parameters are permitted.
  - Actions proceed without restriction, subject to other checks like `SkipCondition` or `PreflightCheck`.

### Example Workflow

Consider a resource with the above configuration and a request with:
- Method: `POST`
- Path: `/api/v1/users`
- Headers: `Content-Type`, `Authorization`
- Query Parameters: `user_id`, `token`

In `APIServerMode`:
- The `RestrictToHTTPMethods` validation fails (`POST` is not in `["GET"]`), so the action is skipped.
- The `RestrictToRoutes` validation would also fail (`/api/v1/users` is not in `["/api/v1/whois"]`).
- The `AllowedHeaders` validation would fail if `request.header("Authorization")` is used, as it's not in `["Content-Type"]`.
- The `AllowedParams` validation would fail if `request.params("token")` is used, as it's not in `["user_id", "session_id"]`.

The action is skipped at the first validation failure, and a log entry details the reason.

### Best Practices

- **Use Specific Restrictions:** Define only the necessary HTTP methods and routes to minimize skipping and ensure intended behavior.
- **Leverage Logging:** Review log messages for skipped actions to diagnose validation issues (e.g., incorrect method or path).
- **Test Configurations:** Validate resource configurations in a test environment to ensure the correct methods, routes, headers, and parameters are permitted.
- **Combine with Preflight Validations:** Use API request validations alongside [Preflight Validations](../workflow-control/validations.md) for comprehensive checks, as they serve complementary purposes.

By incorporating API request validations into your resources, you can enforce strict request compliance, enhance security, and streamline action execution in API-driven workflows.
