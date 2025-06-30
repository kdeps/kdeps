---
outline: deep
---

# Expr Block

The `expr` block is space for evaluating standard PKL expressions. It is primarily used to execute
expressions that produce side effects, such as updating resources or triggering actions, but also supports
general-purpose evaluation of any valid PKL expression, making it a place for inline logic and
scripting within a configuration.

## Overview of the `expr` Block

The `expr` block is designed to evaluate PKL expressions in a straightforward manner. Its key uses include:

- **Side-Effecting Operations**: Executing functions like `memory.setRecord` that modify resources or state without
  returning significant values.

- **Inline Scripting**: Evaluating arbitrary PKL expressions to implement logic, assignments, or procedural tasks
  directly within a configuration.

The `expr` block simplifies the execution of side-effecting operations that does not makes sense to output it's results.

## Syntax and Usage

The `expr` block is defined as follows:

```apl
expr {
  // Valid PKL expression(s)
}
```

Each expression within the block is evaluated in sequence, allowing multiple expressions to form a procedural sequence if needed.

The `expr` block is well-suited for operations that update state, such as setting memory items.

```apl
expr {
  "@(memory.setRecord("status", "active"))"
}
```

In this example, the memory store is updated to indicate an active status. The `memory.setRecord` function is executed as a side effect, and no return value is required. This also applies to `memory.clear()`.
