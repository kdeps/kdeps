# Git component

The `git` component runs version-control operations (status, diff, log, show, branch, remote, add, commit, checkout, init, clone, push, pull) and returns parsed results: file lists, commit objects, branch lists. It is a YAML component from the `components` asset set. Call it with a `component:` resource, no install step. The operations are the kdeps API's `git_ops` tool. The component only POSTs the arguments.

```yaml
# resources/status.yaml
actionId: gitStatus
name: Check Repository Status
component:
  name: git                    # from the components asset set; a components/git/ of your own wins
  with:
    operation: status
    workingDir: /path/to/repo  # default: the current directory
```

## Read next

- [Git component reference](/reference/resources/git) - basic usage, inputs, output, best practices, how it is built
- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [exec component](/workflow/resources/exec) - shell commands (use for other CLI tools)
- [Git component source](https://github.com/kdeps/kdeps/tree/main/assets/components/git) - the component and its tool asset
