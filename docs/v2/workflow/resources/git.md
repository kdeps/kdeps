# Git component

The `git` component runs structured version-control operations - status, diff, log, show, branch, remote, add, commit, checkout, init, clone, push, pull - and returns parsed results (file lists, commit objects, branch lists) instead of raw command output. It is a kdeps component (YAML, no Go code) from the `components` asset set: call it with a `component:` resource, no install step.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode, the workflow containing it runs as a single callable tool.

## Requirements

`git` on `PATH`, and Python through uv like any [python resource](/workflow/resources/python) - the component's single step is a python resource that runs git and prints the structured result.

## Basic usage

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

```yaml
# resources/commit.yaml
actionId: gitCommit
name: Commit Changes
requires: [gitStatus]
component:
  name: git
  with:
    operation: commit
    workingDir: /path/to/repo
    message: "feat: add new feature"
    dryRun: true               # report what would happen, change nothing
```

## Inputs

| Input | Operation | Description |
|---|---|---|
| `operation` | all | `status`, `diff`, `log`, `show`, `branch`, `remote`, `add`, `commit`, `checkout`, `init`, `clone`, `push`, `pull` (required) |
| `workingDir` | all | Repository directory; for `clone`, the target directory |
| `paths` | add, checkout, diff, log | Space-separated paths (`add` defaults to `.`) |
| `message` | commit | Commit message (required for commit) |
| `branch` | checkout, push, pull | Branch name |
| `url` | clone | Repository URL |
| `remote` | push, pull | Remote name (default `origin`) |
| `args` | show | Extra space-separated git arguments, e.g. `HEAD~1` |
| `maxCount` | log | Commits to list (default 10) |
| `dryRun` | add, commit, checkout, init, clone, push, pull | Preview without modifying |

Component inputs are passed as strings, so `paths` and `args` are space-separated (quote a path with spaces, shell-style: `paths: "'my file.txt' src/"`).

## Output

Every result has `success: true` plus the operation's fields; a failing git command fails the resource with git's error.

```json
{"success": true, "branch": "main", "staged": ["file1.go"], "unstaged": ["file2.go"], "untracked": ["new.txt"], "conflicts": []}
```

| Operation | Fields |
|---|---|
| `status` | `branch`, `staged`, `unstaged`, `untracked`, `conflicts` |
| `diff` | `diff`, `additions`, `deletions` |
| `log` | `commits` (`hash`, `author`, `email`, `date`, `message`), `count` |
| `show` | `output` |
| `branch` | `branches` (`name`, `current`, `remote`), `current`, `count` |
| `remote` | `remotes` (`name`, `url`, `type`), `count` |
| `add` / `commit` / `checkout` / `init` / `clone` / `push` / `pull` | `staged` / `committed`+`message` / `switched`+`branch`+`paths` / `initialized`+`path` / `cloned`+`url` / `pushed` or `pulled`+`remote`+`branch`; `dryRun: true` when previewing |

```yaml
# resources/report.yaml
requires: [gitStatus]
apiResponse:
  success: true
  response:
    branch: "{{ output('gitStatus').branch }}"
    untracked: "{{ output('gitStatus').untracked }}"
```

## Best practices

1. **Always set `workingDir`** - avoid ambiguity about which repository is being operated on
2. **Use `dryRun` on write operations** - verify before committing, pushing, or checking out
3. **Use `status` before `add`** to understand what will be staged
4. **Set `maxCount` on `log`** to keep output bounded

## See also

- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [Exec resource](/workflow/resources/exec) - shell commands (use for other CLI tools)
- [Python resource](/workflow/resources/python) - what the component's step runs on
