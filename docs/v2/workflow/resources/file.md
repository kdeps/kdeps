# File component

The `file` component runs one filesystem operation (read, write, patch, list, delete, exists, mkdir, copy, move, append) and returns a structured result.

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

- [File component reference](/reference/resources/file) - every key, option and detail
- [exec component](/workflow/resources/exec) - shell commands
- [Python resource](/workflow/resources/python) - complex data processing
