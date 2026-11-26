# Use conditionals and list operations

Build one resource that demonstrates every control-flow
tool kdeps expressions give you: the ternary operator, the logical operators,
and the list functions `filter`, `map`, `all`, and `any`.

The resource sets values with expressions before it responds:

```yaml
# resources/demo.yaml (part)
before:
  - "set('label', 20 >= 18 ? 'adult' : 'child')"
  - "set('adults', filter(get('people'), {.age >= 18}))"
  - "set('names', map(get('people'), {.name}))"
```

## Read next

- [Full walkthrough](/reference/examples/control-flow) - every step, file and command
- [Expressions](/workflow/expressions) - where expressions run
- [Validation and control flow](/workflow/validation) - `skip` and `check`
