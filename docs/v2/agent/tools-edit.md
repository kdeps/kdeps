# Edit file

`edit_file` changes one line of an existing file: give the line number (or words that are on it) and either replacement text or text to append. It is a [declared python tool](/agent/tools#declared-python-tools) in `tools/edit_file.yaml`, so its whole behavior is in that YAML. Reads, whole-file writes and `match_id` are on [File tools](/agent/tools-files).

```text
read_file (line numbers) -> edit_file {file_path, line | match, text | append} -> edited line(s), numbered
```

## Replace a line

```json
{"file_path": "/repo/app.py", "line": 12, "text": "    timeout = 30"}
```

Line 12 becomes `    timeout = 30`. `text` may span several lines; they replace the one line.

## Replace words on a line

```json
{"file_path": "/repo/app.py", "line": 12, "match": "30", "text": "60"}
```

Only the first `30` on line 12 becomes `60`. Without `line`, the first line containing `match` is used.

## Append

```json
{"file_path": "/repo/app.py", "line": 12, "append": "    retries = 3"}
```

`    retries = 3` is inserted after line 12. With neither `line` nor `match`, it goes after the end of the file.

## Result and errors

The result is the edited line number and the line(s) as they now read:

```text
edited /repo/app.py at line 12:
    12	    timeout = 60
```

A call fails, and the file is left as it was, when: both or neither of `text` and `append` are given; `line` is outside the file; the line does not contain `match`; or no line contains `match`. Empty arguments count as not given, so `text` cannot clear a line to empty. To create a file or rewrite a whole file, use `write_file`.
