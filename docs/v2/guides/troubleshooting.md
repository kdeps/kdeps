# Troubleshooting

Common errors, what they mean, and how to fix them.

## Dependency Errors

### dependency cycle detected

```
Error: dependency cycle detected (code: DEPENDENCY_CYCLE)
```

Two or more resources form a circular `requires` chain (A requires B, B requires A). The engine detects cycles during graph construction and fails before execution.

Fix: break the cycle by removing one of the `requires` edges. If both resources genuinely need each other's output, merge them into a single resource.

### resource 'X' depends on unknown resource 'Y'

```
Error: resource 'fetchData' depends on unknown resource 'missingResource'
```

A `requires` field references an actionId that doesn't exist in the workflow.

Fix: check for typos in the `requires` list, or add the missing resource. ActionIds are case-sensitive.

### target resource 'X' not found

```
Error: target resource 'respond' not found
```

`metadata.targetActionId` references a resource that doesn't exist.

Fix: ensure the actionId in `targetActionId` matches exactly (case-sensitive) one of the resources in the workflow.

## Validation Errors

### Input validation failed

```
Error: Input validation failed (code: VALIDATION)
Details: [{"field": "email", "message": "required field missing"}]
```

A `validations.check` expression evaluated to false.

Fix: check the request input against the check expressions. Use `kdeps run --debug` to see the evaluated expression values.

### Preflight check failed

```
Error: Preflight check failed: validation expression error: ...
```

A check expression contains a syntax error or references an undefined variable.

Fix: validate the expression syntax. Common causes:
- Unclosed `{{ }}` braces
- Misspelled function names (`get` not `Get`, `len` not `length`)
- Referencing a key that doesn't exist without a nil check

## Expression Errors

### Expression evaluation failed

```
Error: Expression evaluation failed (code: EXPRESSION_ERR)
```

An expression in `before:`, `after:`, or `validations` couldn't be evaluated.

Common causes:
- **Type mismatch**: `get('age') > 'old'` (comparing number to string)
- **Nil access**: `get('name').toLowerCase()` when name is nil -- use `get('name') ?? ''` first
- **Undefined function**: using a function that doesn't exist in expr-lang

### Missing value in string interpolation

<div v-pre>

If `{{ get('missingKey') }}` renders as empty string, the key doesn't exist in any data source. Use `default(get('missingKey'), 'fallback')` to provide a default.

</div>

## LLM / Chat Errors

### model not found

```
Error: model 'llama3.2' not found
```

The configured LLM model isn't available on the backend.

Fix:
- For Ollama: run `ollama pull llama3.2` to download the model
- For OpenAI: check the model name (e.g., `gpt-4o`, not `gpt-4`)
- Verify the `--model` flag or `KDEPS_AGENT_MODEL` env var

### Missing API key

```
Error: authentication required
```

The LLM backend requires an API key but none was provided.

Fix: set the appropriate environment variable:
- OpenAI: `OPENAI_API_KEY`
- Anthropic: `ANTHROPIC_API_KEY`
- Or pass via `KDEPS_AGENT_BASE_URL` headers

## HTTP Client Errors

### Resource execution timed out

```
Error: Resource execution timed out after 30s (code: TIMEOUT)
```

The resource's action exceeded its timeout.

Fix: increase the timeout on the resource:
```yaml
httpClient:
  url: https://api.example.com/slow
  timeout: 60s  # increase from default 30s
```

### Resource not reachable

```
Error: connection refused
```

The target URL is not accepting connections.

Fix:
- Verify the URL is correct and the service is running
- Check network connectivity from the kdeps host
- If the service is behind a VPN or firewall, ensure access is allowed

## Python Errors

### Python script failed

```
Error: Resource execution failed: python exited with code 1
```

The Python script returned a non-zero exit code.

Fix: check stderr output. Run the workflow with `--debug` to see full Python output. Common causes:
- Missing Python package (install with `pip`)
- Syntax error in inline script
- Python version incompatibility

## Exec Errors

### Command not found

```
Error: Resource execution failed: exec: "mycommand": executable file not found in $PATH
```

The shell command isn't available on the system.

Fix: install the command or use the full path. Verify the command works in a regular shell first.

## Loop Errors

### Loop exceeded max iterations

```
Error: loop exceeded maxIterations (1000)
```

A `loop.while` condition never became false, and the safety cap stopped execution.

Fix:
- Check the `while` expression -- is it ever becoming false?
- Increase `maxIterations` if you genuinely need more iterations
- Add an `every` delay to slow the loop if it's running too fast

## Debugging

### Enable debug logging

```bash
kdeps run workflow.yaml --debug
```

Shows detailed execution logs including expression evaluation, dependency resolution, and resource dispatch.

### Validate workflow without executing

```bash
kdeps run workflow.yaml --validate
```

Checks the workflow schema, dependency graph, and expression syntax without running any actions.

### Check schema only

```bash
kdeps run workflow.yaml --check-schema
```

Validates the YAML structure matches the expected schema. Faster than `--validate`.

## See Also

- [Execution Flow](/guides/execution-flow) -- how the DAG resolves and runs
- [Validation & Control Flow](/concepts/validation-and-control) -- skip and check logic
- [Expression Functions Reference](/reference/expression-functions-reference) -- all available functions
- [CLI Reference](/reference/cli/) -- all flags and commands
