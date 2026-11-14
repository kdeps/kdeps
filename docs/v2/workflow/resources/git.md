# Git component

The `git` component runs version-control operations (status, diff, log, show, branch, remote, add, commit, checkout, init, clone, push, pull) and returns parsed results: file lists, commit objects, branch lists. It is a YAML component from the `components` asset set. Call it with a `component:` resource, no install step. The operations are the kdeps API's `git_ops` tool. The component only POSTs the arguments.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode, the workflow containing it runs as a single callable tool.

## Requirements

`git` and `python3` (or `python`) on `PATH`. The operations are the `git_ops` tool. This tool is a `python:` block in `tools/git_ops.yaml`. The kdeps API serves it in the kdeps process. The component makes one call and returns the result.

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
flow:
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

Component inputs are strings. For this reason, spaces separate the items in `paths` and `args`. Put a path with spaces in quotes, as in a shell: `paths: "'my file.txt' src/"`.

## Output

Every result has `success: true` plus the operation's fields. A failing git command fails the resource with git's error.

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
flow:
  # resources/report.yaml
  requires: [gitStatus]
apiResponse:
  success: true
  response:
    branch: "{{ output('gitStatus').branch }}"
    untracked: "{{ output('gitStatus').untracked }}"
```

## Best practices

1. **Always set `workingDir`** - then it is clear which repository the operation uses
2. **Use `dryRun` on write operations** - verify before committing, pushing, or checking out
3. **Use `status` before `add`** to see the changes that `add` stages
4. **Set `maxCount` on `log`** to keep output bounded

## How it is built

```text
component: git -> POST /apis/tools.kdeps.io/v1/tools/git_ops/call -> {success, ...operation fields}
```

One `httpClient` POST carrying the ten inputs to the `git_ops` tool, then an `interface.output` that decodes the tool's JSON body. The tool runs `git` with `GIT_PAGER=cat` and `GIT_TERMINAL_PROMPT=0` and parses the output into the same fields. A command that fails returns `git <operation> failed: exit status N` with the stderr of git. Before git runs, kdeps checks `workingDir` against `KDEPS_WORKSPACE_ROOT`. kdeps permits paths under `~/.kdeps` and `/tmp`. kdeps stops one call after 5 minutes.

To change an input default, copy `components/git/` into your project. To change the operations, copy `tools/git_ops.yaml` to `~/.kdeps/tools/`.

## See also

- [Component resource](/agencies/component-resource) - how `component:` calls pass `with:` inputs
- [exec component](/workflow/resources/exec) - shell commands (use for other CLI tools)
- [Git component source](https://github.com/kdeps/kdeps/tree/main/assets/components/git) - the component and its tool asset
