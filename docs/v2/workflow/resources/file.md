# File component

The `file` component runs one filesystem operation - read, write, patch, list, delete, exists, mkdir, copy, move, append - and returns a structured result (content and lines, entry lists, flags) instead of raw command output. It is a kdeps component (YAML, no Go code) from the `components` asset set: call it with a `component:` resource, no install step. Its operations are the kdeps API's `file_ops` tool; the component only POSTs the arguments.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode, the workflow containing it runs as a single callable tool.

## Requirements

None. The operations are served by the kdeps API's `file_ops` tool in the kdeps process; the component makes one call and returns the result.

## Basic usage

<div v-pre>

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

```yaml
# resources/write.yaml
actionId: writeReport
name: Write Report
requires: [readConfig]
component:
  name: file
  with:
    operation: write
    path: out/report.txt       # parent directories are created
    content: "{{ output('readConfig').content }}"
    backup: true               # an existing out/report.txt is copied to out/report.txt.bak first
```

</div>

## Inputs

| Input | Operation | Description |
|---|---|---|
| `operation` | all | `read`, `write`, `patch`, `list`, `delete`, `exists`, `mkdir`, `copy`, `move`, `append` (required) |
| `path` | all | Target file or directory; the destination for `copy` and `move` |
| `source` | copy, move | Source path |
| `content` | write, append | Text to write or append |
| `patch` | patch | Unified diff to apply |
| `encoding` | read | `text` (default) or `base64` |
| `pattern` | list | Glob on entry names, e.g. `*.go` (case-sensitive) |
| `recursive` | list | Walk subdirectories |
| `backup` | write, patch | Copy the existing file to `<path>.bak` first |
| `dryRun` | write, patch, delete, mkdir, copy, move, append | Report what would happen, change nothing |
| `mode` | write, mkdir | Octal permissions; default `0644` (write), `0755` (mkdir) |
| `appendNewline` | write, append | End the content with a newline |

## Output

Every result has `success` plus the operation's fields. A failing operation (missing file, patch that does not apply, missing required input) fails the resource with a `file: ...` error. Two operations report instead of failing: `delete` of a missing path returns `deleted: false, reason: not_found`, and `exists` returns `success: false` when the path is missing.

| Operation | Fields |
|---|---|
| `read` | `content`, `encoding`, `path`, `exists`, `size` (bytes), `lines` |
| `write` | `path`, `size`, `written`, `mode`, `exists` (before the write), `backup`, `backupPath` |
| `patch` | `path`, `patched`, `patchLines` |
| `list` | `path`, `entries` (`name`, `path`, `isDir`, `size`, `mode`, `modTime`), `count` |
| `delete` | `path`, `deleted` |
| `exists` | `exists`, `path`, and when it exists `isDir`, `size`, `mode`, `modTime` |
| `mkdir` | `path`, `created`, `mode` |
| `copy` | `source`, `dest`, `copied` |
| `move` | `source`, `dest`, `moved` |
| `append` | `path`, `appended`, `size` |

A dry run returns `dryRun: true` and the flag (`written`, `patched`, `deleted`, ...) as `false`.

```json
{
  "success": true,
  "content": "line1\nline2\n",
  "encoding": "text",
  "path": "notes.txt",
  "exists": true,
  "size": 12,
  "lines": ["line1", "line2", ""]
}
```

`modTime` is ISO 8601 with the local offset, e.g. `2026-10-14T10:00:00+02:00`.

## Patch

`patch` applies a unified diff: `@@ -N,M +N,M @@` hunk headers, context lines (` `), removals (`-`), additions (`+`). A context or removed line that does not match the file fails the resource and leaves the file unchanged.

<div v-pre>

```yaml
# resources/fix.yaml
actionId: applyFix
name: Apply the fix
requires: [suggestFix]
component:
  name: file
  with:
    operation: patch
    path: main.go
    patch: "{{ output('suggestFix').patch }}"   # the diff an LLM step produced
    backup: true                                # main.go.bak keeps the original
    dryRun: false                               # true: check the patch applies, write nothing
```

</div>

## List

```yaml
# resources/list.yaml
actionId: goFiles
name: List Go files
component:
  name: file
  with:
    operation: list
    path: src
    pattern: "*.go"            # matched against entry names, not full paths
    recursive: true            # entries from every subdirectory, sorted by path
```

Listing a file (not a directory) returns one entry for it.

## Errors

| Error | Cause |
|---|---|
| `file: path is required for read operation` (and the other operations) | `path` is empty |
| `file: source is required for copy operation` | `source` is empty for copy or move |
| `file: patch failed: patch context mismatch: ...` | The diff does not match the file |
| `file: cannot access path: ...` | `list` on a path that does not exist |
| `file: source does not exist: ...` | `move` from a missing path |
| `file: unsupported operation 'x'` | `operation` is not one of the ten |

## How it is built

```text
component: file -> POST /apis/tools.kdeps.io/v1/tools/file_ops/call -> {success, ...operation fields}
```

One `httpClient` POST carrying all twelve inputs to the `file_ops` tool, then an `interface.output` that decodes the tool's JSON body. The tool does each operation with the Go standard library (`os`, `path/filepath`) and returns the same fields. Copy `components/file/` into your project to change an input default; the operations themselves live in the kdeps API.

## See also

- [exec component](/workflow/resources/exec) - shell commands
- [Python resource](/workflow/resources/python) - complex data processing
- [Git component](/workflow/resources/git) - version control operations
