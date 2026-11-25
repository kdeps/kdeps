# Expressions

Expressions pass data between resources, validate inputs, and run conditional logic, using [expr-lang](https://expr-lang.org/). They run across workflow resources.

<div v-pre>

```yaml
# resources/example.yaml
chat:
  context:
    prompt: "Hello {{ get('name') }}, today is {{ info('timestamp') }}"
```

</div>

## Read next

- [Expressions reference](/reference/workflow/expressions) - every key, option and detail
