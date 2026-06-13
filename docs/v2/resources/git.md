# Git Resource

The `git:` resource provides structured version control operations for reading repository state and making commits. It replaces ad-hoc shell commands (`exec git status`, `exec git log`) with type-safe, composable DAG operations that return structured output.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In workflow mode it executes as a DAG step. In agent mode, the workflow containing this resource runs as a single callable tool.

## Basic Usage

```yaml
# resources/status.yaml
actionId: gitStatus
name: Check Repository Status
git:
  operation: status
  workingDir: /path/to/repo
```

```yaml
# resources/commit.yaml
actionId: gitCommit
name: Commit Changes
git:
  operation: commit
  workingDir: /path/to/repo
  message: "feat: add new feature"
  dryRun: true
```

## Operations

| Operation | Description |
|-----------|-------------|
| `status` | Show working tree status (`--porcelain -b`) |
| `diff` | Show unstaged changes |
| `log` | Show commit history |
| `show` | Show commit details |
| `branch` | List branches |
| `remote` | List remote repositories |
| `add` | Stage files |
| `commit` | Create a commit |
| `checkout` | Switch branches or restore files |
| `init` | Initialize a new repository |
| `clone` | Clone a repository |
| `push` | Push to remote |
| `pull` | Pull from remote |

## Configuration Options

| Option | Operation | Description |
|--------|-----------|-------------|
| `operation` | all | The git operation to perform (required) |
| `workingDir` | all | Working directory for git commands |
| `paths` | add, checkout, diff, log | File paths to operate on |
| `message` | commit | Commit message |
| `branch` | checkout, branch, push, pull | Branch name |
| `url` | clone | Remote repository URL |
| `remote` | push, pull | Remote name (default: `origin`) |
| `args` | show, checkout | Additional git arguments |
| `maxCount` | log | Maximum commits to show (default: 10) |
| `dryRun` | add, commit, checkout, init, clone, push, pull | Preview without modifying |
| `format` | log, show | Custom git format string |

## Operation Details

### Status

Returns structured output with branch, staged, unstaged, untracked, and conflicted files:

```yaml
# resources/status.yaml
git:
  operation: status
  workingDir: /path/to/repo
```

**Output:**
```json
{
  "success": true,
  "branch": "main",
  "staged": ["file1.go"],
  "unstaged": ["file2.go"],
  "untracked": ["new.txt"],
  "conflicts": []
}
```

### Log

Returns structured commit history:

```yaml
# resources/log.yaml
git:
  operation: log
  workingDir: /path/to/repo
  maxCount: 5
  paths: ["src/"]
```

**Output:**
```json
{
  "success": true,
  "commits": [
    {"hash": "abc123", "author": "Alice", "email": "alice@example.com", "date": "2026-06-13", "message": "fix: bug"}
  ],
  "count": 1
}
```

### Diff

Returns the unified diff with addition/deletion counts:

```yaml
# resources/diff.yaml
git:
  operation: diff
  workingDir: /path/to/repo
```

### Commit

Creates a commit with the given message:

```yaml
# resources/commit.yaml
git:
  operation: commit
  workingDir: /path/to/repo
  message: "feat: implement search"
```

### Add

Stages files for commit:

```yaml
# resources/add.yaml
git:
  operation: add
  workingDir: /path/to/repo
  paths: ["src/main.go", "src/utils.go"]
```

## Dry Run Mode

Every write operation supports `dryRun: true` to preview what would happen without modifying the repository:

```yaml
# resources/dry-run-commit.yaml
git:
  operation: commit
  message: "feat: dangerous change"
  dryRun: true
```

## Best Practices

1. **Always set `workingDir`** - avoid ambiguity about which repository is being operated on
2. **Use `dryRun` on write operations** - verify before committing, pushing, or checking out
3. **Check `success` field** in downstream resources to detect git failures
4. **Use `status` before `add`** to understand what will be staged
5. **Set `maxCount` on `log`** to avoid unbounded output

## See Also

- [Exec Resource](exec) - shell commands (use for non-git CLI tools)
- [Workflow Configuration](../configuration/workflow) - settings reference
