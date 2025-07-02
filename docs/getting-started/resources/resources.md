---
outline: deep
---

# Resources

Resources are the fundamental units of action in an AI agent and serve as the workhorses of Kdeps. Each resource is
designed to perform a specific task, making them essential for executing workflows efficiently.

## Key Characteristics

- Resources are stored as `.pkl` files in the `resources/` directory. For more information about Apple PKL, visit the
  [Apple PKL Official Website](https://pkl-lang.org/).

- They enable the AI agent to perform diverse operations, such as running scripts, processing API requests, and
  interacting with language models. See [Kdeps Resource Types](../resources/types.md)

- Resources can be remixed and reused by other Kdeps AI Agents.

- They form a dependency graph that determines the execution order of the workflow.

Similar to [Workflows](../configuration/workflow.md), resources have their own configurations. These configurations
define the behavior, dependencies, and validation logic for each resource.

## Common Resource Configurations

- **Metadata**:
   - **`ActionID`**: A unique identifier for the resource.
   - **`Name`**: A human-readable name for the resource.
   - **`Description`**: A brief explanation of what the resource does.
   - **`Category`**: A classification for organizing resources.

- **Dependencies**:
   - **`Requires`**: Specifies the dependencies of the resource. This ensures the resource executes only after its
     dependencies are satisfied. See [Graph Dependency](../resources/kartographer.md) for more information.

- **Multiple Items Iterations**:
   - **`Items`**: Specify multiple items to be iterated over in a loop. Values can be obtained via `item.current()`,
     `item.prev()`, and `item.next()`.

### **Execution Logic**

The `Run` block defines the execution logic for a resource, including conditional execution, validation checks, and request-level constraints. This section is relevant when `APIServerMode` is enabled.

#### **Key Fields:**

- **`SkipCondition`**
  Specifies one or more conditions under which the resource execution should be skipped. If any condition evaluates to `true`, the resource is bypassed.
  See [Skip Conditions](../resources/skip.md).

- **`PreflightCheck`**
  Performs validation before execution begins. If validation fails, execution is aborted and a custom error is returned.
  See [Preflight Validations](../resources/validations.md).

  - **`Validations`**: A list of boolean expressions. If any expression evaluates to `false`, the check fails.
  - **`Error`**:
    - **`Code`**: HTTP status code to return (e.g., `404`)
    - **`Message`**: Error message included in the response

- **`API Request Validations`**
  These validations are enforced only in `APIServerMode`. If any validation fails, the action is skipped
  entirely—meaning no further steps such as `Exec`, `Python`, `Chat`, or `HTTPClient` will run. If any field is left
  empty, it defaults to allowing all values for that category.

  For more information, please visit [API Request Validations](../resources/api-request-validations.md).

  - **`RestrictToHTTPMethods`**:
    Limits which HTTP methods (e.g., `GET`, `POST`) are allowed.

  - **`RestrictToRoutes`**:
    Limits which URL paths (e.g., `/api/v1/whois`) the request must match.

  - **`AllowedHeaders`**:
    Specifies which HTTP headers are allowed in the request.

  - **`AllowedParams`**:
    Specifies which query parameters are permitted in the request.
