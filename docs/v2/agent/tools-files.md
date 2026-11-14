# File tools

`read_file`, `write_file`, `edit_file`, `list_files`, `md5_file`, and `tail_file` run in the agent REPL with no setup and no API key. All six are `python:` blocks in their tool assets (see [Built-in tools](/agent/tools#tool-definitions)) and run on the system python. Before the tool runs, kdeps compares each path argument with `KDEPS_WORKSPACE_ROOT` and the root-directory guard.

kdeps permits paths under `~/.kdeps` and `/tmp` with no prompt, also when the workspace root is a different folder. On Windows, the permitted temp directory is the process temp directory. kdeps blocks a directory that has only the same name, and it always blocks `/`. kdeps converts a relative path to the absolute path in the working directory. Then it checks that path and gives it to the script.

Permission modes, lean mode, the handshake, and harness reminders stay on [Built-in tools](/agent/tools).

Each of these tools is also a workflow component. The component POSTs to `/apis/tools.kdeps.io/v1/tools/{name}/call` and passes `file_path` (`path` for `list_files`). The session's last file, `match_id`, and the colored diff stay in the agent loop.

## File operations

Always available. No environment variables required.

| Tool | Description |
|------|-------------|
| `read_file` | Read a text file with a 1-based line number on every line. PDF, DOCX, EPUB, RTF and ODT are refused with a pointer to `load_document`, which extracts their text |
| `write_file` | Write or overwrite a file with the exact bytes given (no newline translation), then read it back to verify |
| `edit_file` | Replace or append on one line: `line` or `match`, then `text` or `append` |
| `list_files` | List directory contents |
| `md5_file` | Compute a file's MD5 hash - cheap way to check whether content actually changed |
| `tail_file` | Read the last N lines of a file without loading the whole thing |

## Line numbers

`read_file` prefixes every line with its 1-based number and renders it `cat -A` style: tabs as `^I`, other control characters as `^X`, and `$` at the true line end (`  42⇥code$`). Indentation, trailing whitespace and stray control characters stay visible. `edit_file`'s `line` uses these numbers.

## Size cap

`read_file` refuses a file, and `write_file` refuses content, larger than the `file-read-limit` event (1 MiB by default). The agent loop passes the event's value to the script; a workflow component that calls the tool gets the 1 MiB default.

```yaml
# ~/.kdeps/events/file-read-limit.yaml
name: file-read-limit
on:
  bytes: 4194304  # read_file and write_file refuse anything past 4 MiB
run: reject
```

## Optional file_path

`read_file` treats `file_path` as optional: omit it and it reads the file most recently read, edited, or written this session. This helps when the model means "the file that I read last" and calls `read_file` with only `offset`/`limit`. `write_file`, `edit_file`, `md5_file`, and `tail_file` always need an explicit path.

## Colored diff

`write_file` and `edit_file` print a colored diff under the tool call: removed lines red, added green, with a few context lines. kdeps shortens large diffs (for example, a full new file). The diff shows only in the terminal. The model gets a short result.

## Jumping from search_local to read_file with match_id

A `search_local` result that finds its query in a file carries `match_id`, `line` and `revision` besides `path`/`snippet`. Pass `match_id` to `read_file` instead of `file_path` to jump to that hit. `context_before`/`context_after` (default 20 each, like `edit_file`'s `anchor` view) set how much code comes back:

```xml
<invoke name="search_local">
  <parameter name="path">/app</parameter>
  <parameter name="query">func handleLogin</parameter>
</invoke>
```

```json
{"results": [{"path": "/app/server.go", "match_id": "match-4a9f21bc", "line": 112, "revision": "sha256:...", "snippet": "..."}]}
```

```xml
<invoke name="read_file">
  <parameter name="match_id">match-4a9f21bc</parameter>
  <parameter name="context_after">40</parameter>
</invoke>
```

A `match_id` is local to the process. kdeps keeps a maximum number of them and removes the oldest first. Use it to connect a search to a read in the same session. It is not a permanent reference. If kdeps cannot find a `match_id` (removed, or from a different process), it gives a clear error with the ID. An explicit `file_path` always has priority over `match_id`.

Line edits with `edit_file` are on [Edit file](/agent/tools-edit).
