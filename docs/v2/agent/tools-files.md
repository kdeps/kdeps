# File tools

`read_file`, `write_file`, `edit_file`, `list_files`, `md5_file`, and `tail_file` run in the agent REPL with no setup and no API key. All six are `python:` blocks in their tool assets (see [Built-in tools](/agent/tools#tool-definitions)) and run on the system python. Every path argument is checked against `KDEPS_WORKSPACE_ROOT` and the root-directory guard before the tool runs; a relative path is checked, and handed to the script, as the absolute path it names in the working directory. Permission modes, lean mode, the handshake, and harness reminders stay on [Built-in tools](/agent/tools).

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

Every line `read_file` returns is prefixed with its 1-based line number, and the line itself is rendered `cat -A` style: tabs show as `^I`, other control characters as `^X`, and a `$` marks the true end of the line (`  42⇥code$`) - so indentation, trailing whitespace, and stray control characters are visible instead of silently hidden. Those line numbers are what `edit_file`'s `line` points at.

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

`read_file` treats `file_path` as optional: omit it and it reads the file most recently read, edited, or written this session. This covers the common slip where the model means "the file I was just looking at" and calls `read_file` with only `offset`/`limit`. `write_file`, `edit_file`, `md5_file` and `tail_file` always require an explicit path.

## Colored diff

`write_file` and `edit_file` print a colored diff of what changed under the tool call - removed lines in red, added lines in green, with a couple of context lines - so you can see every change the agent makes at a glance. Large diffs (e.g. writing a whole new file) are capped. The diff is shown in the terminal only; the model receives a concise result, not the ANSI-colored text.

## Jumping from search_local to read_file with match_id

Every `search_local` result that finds its query in the file carries `match_id`, `line`, and `revision` alongside the usual `path`/`snippet` fields. Pass `match_id` to `read_file` instead of `file_path` to jump straight to that hit - `context_before`/`context_after` (default 20 each) control how much surrounding code comes back, the same defaults `edit_file`'s `anchor` view uses:

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

A `match_id` is process-local and capped in count (oldest evicted first) - it is for chaining a search into a read within the same session, not a durable reference. An unresolvable one (evicted, or from a different process) is a clear error naming it; `file_path` given explicitly always takes priority over `match_id`.

Line edits with `edit_file` are on [Edit file](/agent/tools-edit).
