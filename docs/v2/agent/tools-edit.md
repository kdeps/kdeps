# Edit file

`edit_file` changes one line of an existing file. Give the line number (or words that are on the line). Then give the replacement text or the text to append. It is a [declared python tool](/agent/tools#declared-python-tools) in `tools/edit_file.yaml`. That YAML file contains all of its behavior. Reads, whole-file writes and `match_id` are on [File tools](/agent/tools-files).

```text
read_file (line numbers) -> edit_file {file_path, line | match, text | append} -> edited line(s), numbered
```

## Replace a line

```json
{"file_path": "/repo/app.py", "line": 12, "text": "    timeout = 30"}
```

Line 12 becomes `    timeout = 30`. `text` can have several lines. They replace the one line.

## Replace words on a line

```json
{"file_path": "/repo/app.py", "line": 12, "match": "30", "text": "60"}
```

Only the first `30` on line 12 becomes `60`. Without `line`, the tool uses the first line that contains `match`.

## Append

```json
{"file_path": "/repo/app.py", "line": 12, "append": "    retries = 3"}
```

The tool inserts `    retries = 3` after line 12. If you give no `line` and no `match`, the text goes after the end of the file.

## Result and errors

The result is the edited line number and the line(s) as they now read:

```text
edited /repo/app.py at line 12:
    12	    timeout = 60
```

A call fails and leaves the file unchanged when:

- you give both `text` and `append`, or neither of them (an empty value is the same as no value, so `text` cannot clear a line)
- `line` is outside the file, or the line does not contain `match`
- no line contains `match`

To create or rewrite a whole file, use `write_file`.
