# File component

The `file` component runs one filesystem operation (read, write, patch, list, delete, exists, mkdir, copy, move, append) and returns a structured result: content and lines, entry lists, flags. It is a YAML component from the `components` asset set. Call it with a `component:` resource, no install step. The operations are the kdeps API's `file_ops` tool. The component only POSTs the arguments.

```yaml
# resources/read.yaml
actionId: readConfig
name: Read Configuration
component:
  name: file                   # from the components asset set; a components/file/ of your own wins
  with:
    operation: read
    path: config.json          # relative paths resolve against the working directory
```

## Read next

- [File component reference](/reference/resources/file) - basic usage, inputs, output, patch, list, errors, how it is built
- [exec component](/workflow/resources/exec) - shell commands
- [Python resource](/workflow/resources/python) - complex data processing
- [Git component](/workflow/resources/git) - version control operations
