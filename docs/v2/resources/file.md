# File Resource

The `file:` resource provides structured filesystem operations for reading, writing, patching, listing, deleting, copying, moving, and appending files and directories. It replaces ad-hoc shell commands (`exec cat`, `exec sed -i`, etc.) with type-safe, composable DAG operations that return structured output.

## Where it runs

Both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In workflow mode it executes as a DAG step. In agent mode, the workflow containing this resource runs as a single callable tool.

## Basic Usage

```yaml
# resources/read.yaml
actionId: readFile
name: Read Configuration
file:
  operation: read
  path: /etc/config.json
```

```yaml
# resources/write.yaml
actionId: writeFile
name: Write Output
file:
  operation: write
  path: /tmp/output.txt
  content: "Hello, World!"
```

## Operations

| Operation | Description |
|-----------|-------------|
| `read` | Read file contents (text or base64) |
| `write` | Write content to a file, creating parent directories |
| `patch` | Apply a unified diff patch to a file |
| `list` | List directory entries, optionally with glob filter |
| `delete` | Delete a file or directory |
| `exists` | Check if a file or directory exists |
| `mkdir` | Create a directory (including parents) |
| `copy` | Copy a file or directory |
| `move` | Move/rename a file or directory |
| `append` | Append content to a file |

## Configuration Options

| Option | Operation | Description |
|--------|-----------|-------------|
| `operation` | all | The file operation to perform (required) |
| `path` | all | Target file or directory path (required except `patch`) |
| `source` | copy, move | Source path for the operation |
| `content` | write, append | Content to write or append |
| `patch` | patch | Unified diff string to apply |
| `encoding` | read | Output encoding: `"text"` (default) or `"base64"` |
| `pattern` | list | Glob pattern to filter results (e.g. `"*.go"`) |
| `recursive` | list | Recurse into subdirectories |
| `backup` | write, patch | Create a `.bak` copy before overwriting |
| `dryRun` | write, patch, delete, mkdir, copy, move, append | Preview without modifying |
| `mode` | write, mkdir | File mode (e.g. `"0644"`, `"0755"`) |
| `appendNewline` | write, append | Ensure content ends with a trailing newline |

## Operation Details

### Read

Reads a file and returns its content. By default returns UTF-8 text. Set `encoding: base64` for binary files.

```yaml
# resources/read.yaml
file:
  operation: read
  path: /path/to/file.txt
  encoding: text
```

**Output:**
```json
{
  "success": true,
  "content": "file contents...",
  "encoding": "text",
  "path": "/path/to/file.txt",
  "exists": true,
  "size": 1234,
  "lines": ["line1", "line2"]
}
```

### Write

Writes content to a file. Creates parent directories automatically. Optionally creates a `.bak` backup if the file already exists.

```yaml
# resources/write.yaml
file:
  operation: write
  path: /tmp/output.txt
  content: "Hello, World!"
  backup: true
  mode: "0644"
```

### Patch

Applies a [unified diff](https://en.wikipedia.org/wiki/Diff#Unified_format) to a file. Supports standard `@@ -N,M +N,M @@` hunk headers, context lines, additions, and removals.

```yaml
# resources/patch.yaml
file:
  operation: patch
  path: /path/to/file.txt
  patch: |
    @@ -1,3 +1,3 @@
     line1
    -line2
    +modified2
     line3
  backup: true
  dryRun: false
```

**Tip:** Use `dryRun: true` to verify the patch applies cleanly before modifying the file.

### List

Lists entries in a directory, or returns single-file info. Supports glob filtering and recursive traversal.

```yaml
# resources/list.yaml
file:
  operation: list
  path: /path/to/dir
  pattern: "*.go"
  recursive: false
```

### Delete

Removes a file or directory. Returns success (not an error) if the path does not exist.

```yaml
# resources/delete.yaml
file:
  operation: delete
  path: /path/to/file.txt
  dryRun: false
```

### Exists

Checks whether a path exists and returns metadata if it does.

```yaml
# resources/exists.yaml
file:
  operation: exists
  path: /path/to/file.txt
```

**Output:**
```json
{
  "success": true,
  "exists": true,
  "path": "/path/to/file.txt",
  "isDir": false,
  "size": 1234,
  "mode": "-rw-r--r--",
  "modTime": "2026-06-13 10:00:00 +0000 UTC"
}
```

### Mkdir

Creates a directory, including all parent directories. Succeeds silently if the directory already exists.

```yaml
# resources/mkdir.yaml
file:
  operation: mkdir
  path: /tmp/nested/dir/structure
  mode: "0755"
```

### Copy

Copies a file or directory to a destination path. Directories are copied recursively.

```yaml
# resources/copy.yaml
file:
  operation: copy
  source: /path/to/source.txt
  path: /path/to/dest.txt
```

### Move

Moves or renames a file or directory. Works across filesystems on supported platforms.

```yaml
# resources/move.yaml
file:
  operation: move
  source: /path/to/source.txt
  path: /path/to/dest.txt
```

### Append

Appends content to the end of a file, creating it if it doesn't exist.

```yaml
# resources/append.yaml
file:
  operation: append
  path: /tmp/log.txt
  content: "new log entry"
  appendNewline: true
```

## Coding Agent Example

The `file:` resource is designed for coding agent workflows. A typical edit cycle:

```yaml
# resources/coding-agent.yaml
# Step 1: Read the file
- actionId: readFile
  name: Read source
  file:
    operation: read
    path: /project/main.go

# Step 2: LLM generates a fix
- actionId: analyzeCode
  name: Analyze and suggest fix
  requires: [readFile]
  chat:
    model: claude-sonnet-4-20250514
    prompt: "Review this code and suggest a fix: {{readFile.content}}"

# Step 3: Apply the fix as a patch
- actionId: applyFix
  name: Apply the fix
  requires: [analyzeCode]
  file:
    operation: patch
    path: /project/main.go
    patch: "{{analyzeCode.patch}}"
    backup: true
```

## Expressions with File Output

The `file:` resource output is a map, so you access fields by key:

```yaml
# Access read content
- set('content', get('readFile').content)
- set('fileSize', get('readFile').size)
- set('lineCount', len(get('readFile').lines))

# Check operation result
- set('writeSucceeded', get('writeFile').success)
- set('wasDryRun', get('writeFile').dryRun)
```

## Dry Run Mode

Every mutating operation (`write`, `patch`, `delete`, `mkdir`, `copy`, `move`, `append`) supports `dryRun: true` to preview what would happen without modifying the filesystem. Use it for safety checks before applying changes:

```yaml
# resources/dry-run-example.yaml
file:
  operation: write
  path: /sensitive/file.txt
  content: "new content"
  dryRun: true
  # Output: { "success": true, "dryRun": true, "written": false }
```

## Backup Safety

Enable `backup: true` on `write` and `patch` operations to automatically create a `.bak` copy before overwriting:

```yaml
# resources/backup-example.yaml
file:
  operation: write
  path: /config/settings.yaml
  content: "new: config"
  backup: true
```

If `/config/settings.yaml` exists, the original is saved to `/config/settings.yaml.bak` before the write.

## Best Practices

1. **Use `file:` instead of `exec cat`/`exec sed`** — structured output, error handling, and DAG composability
2. **Enable backups** on critical files (`backup: true`)
3. **Dry-run before destructive operations** to verify paths and content
4. **Use `patch` for surgical edits** instead of rewriting entire files
5. **Check `exists` before delete** or rely on the idempotent delete behavior
6. **Handle the `success` field** in downstream resources to detect failures

## See Also

- [Exec Resource](exec) — shell commands (use for CLI tools that lack a native resource)
- [Python Resource](python) — complex data processing
- [Git Resource](git) — version control operations
- [Code Intelligence Resource](codeintelligence) — code navigation and search
- [Workflow Configuration](../configuration/workflow)
