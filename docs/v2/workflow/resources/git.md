# Git component

The `git` component runs version-control operations (status, diff, log, show, branch, remote, add, commit, checkout, init, clone, push, pull) and returns parsed results.

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

- [Git component reference](/reference/resources/git) - every key, option and detail
- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [exec component](/workflow/resources/exec) - shell commands (use for other CLI tools)
