---
outline: deep
---

# Exec Resource

The `exec` resource is designed to run shell scripts and commands from the workflow. This is also the resource which
allows accessing the installed Ubuntu packages defined in the `workflow.pkl`. For more information, see the
[Workflow](../configuration/workflow.md) documentation.

## Creating a New Exec Resource

To create a new `exec` resource, you can either generate a new AI agent using the `kdeps new` command or `scaffold` the
resource directly.

Here’s how to scaffold an `exec` resource:

```bash
kdeps scaffold [aiagent] exec
```

This command will add an `exec` resource into the `aiagent/resources` folder, generating the following folder structure:

```bash
aiagent
└── resources
    └── exec.pkl
```

The file includes essential metadata and common configurations, such as [Skip Conditions](../resources/skip) and
[Preflight Validations](../resources/validations). For more details, refer to the [Common Resource
Configurations](../resources/resources#common-resource-configurations) documentation.

## Exec Block

Within the file, you’ll find the `exec` block, which is structured as follows:

```apl
exec {
    Command = """
    echo "hello world"
    """
    Env {
        // Environment variables accessible within the shell
        ["ENVVAR"] = "XYZ"  // Example environment variable
    }
    // Specifies the timeout duration (in seconds) for the shell execution
    TimeoutDuration = 60.s
}
```

Key elements of the `exec` block includes:

- **`Command`**: Specifies the shell command(s) to execute, enclosed in triple double-quotes (`"""`) for multi-line support.
- **`Env`**: Defines environment variables to be available during execution.
- **`TimeoutDuration`**: Determines the exectuion timeout in s (seconds), min (minutes), etc., after which the shell command will be terminated.

When the resource is executed, you can leverage Exec functions like `exec.stdout("id")` to access the output. For
further details, refer to the [Exec Functions](../resources/functions.md#exec-resource-functions) documentation.
