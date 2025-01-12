---
outline: deep
---

# Preflight Validations

Preflight validations are a crucial mechanism for verifying that all required conditions are satisfied before executing
a resource.

These checks ensure that the resource performs as intended, safeguard system integrity, and streamline workflows by
identifying potential issues early in the process.

Preflight validations operate using an `AND` logic, meaning all specified conditions must be met for execution to
proceed.

Additionally, it can accept either a string value of `"true"` (case-insensitive) or a Boolean `true`. Any other input
will result in a Boolean `false`.

## Why Preflight Validations Matter

- **Prevent Issues Early:** By running validations before resource execution, you can catch errors and inconsistencies
  before they cause downstream problems.

- **Custom Error Handling:** Preflight validations enable you to define and return custom error messages, making it
  easier to diagnose and address issues.

## Defining Preflight Validations

Preflight validations are executed before the resource proceeds. They ensure that the resource will only run when
specific criteria are satisfied. Here's an example of how to implement custom validations:

```apl
local OCROutputFile = """
@(read?("file:/tmp/ocrOutput.txt")?.text)
"""

preflightCheck {
    validations {
        OCROutputFile.length != 0
    }

    error {
        code = 422
        message = "The LLM model cannot parse this input, it is empty!"
    }
}
```

This approach ensures that the resource halts execution if the validation fails, providing immediate feedback with a
meaningful error message.

By incorporating preflight validations into your workflow, you can enhance reliability and deliver a better experience
to users or systems relying on your resources.
