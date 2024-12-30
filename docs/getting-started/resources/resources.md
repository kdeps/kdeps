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
   - **`id`**: A unique identifier for the resource.
   - **`name`**: A human-readable name for the resource.
   - **`description`**: A brief explanation of what the resource does.
   - **`category`**: A classification for organizing resources.

- **Dependencies**:
   - **`requires`**: Specifies the dependencies of the resource. This ensures the resource executes only after its
     dependencies are satisfied. See [Graph Dependency](../resources/kartographer.md) for more information.

- **Execution Logic**:
   - **`run`**: Defines the execution logic for the resource, including conditions that affect its behavior:
     - **`skipCondition`**: Specifies conditions under which the resource execution is skipped. If any condition
       evaluates to `true`, the resource will be bypassed. See [Skip Conditions](../resources/skip.md).
     - **`preflightCheck`**: Performs a pre-execution validation and returns a custom error if the validation fails.
       See [Preflight Validations](../resources/validations.md).
       - **`validations`**: Contains validation logic. If any condition evaluates to `false`, an exception is triggered.
       - **`error`**: Defines a custom error returned upon validation failure, with the following attributes:
         - **`code`**: The HTTP error code to return (e.g., `404`).
         - **`message`**: The HTTP error message included in the response.
