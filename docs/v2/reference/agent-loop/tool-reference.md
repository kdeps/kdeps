# Tool reference

Overview: [Built-in tools](/workflow/agent-loop/tools).


The built-in tools of a chat with `loop:`, by group. The catalog and permission modes are on [Built-in tools](/workflow/agent-loop/tools).

## File tools

`read_file`, `write_file`, `edit_file`, `list_files`, `md5_file`, and `tail_file` run in the agent loop with no setup and no API key. All six are `python:` blocks in their tool assets (see [Built-in tools](/reference/agent-loop/tools#tool-definitions)) and run on the system python. Before the tool runs, kdeps compares each path argument with `KDEPS_WORKSPACE_ROOT` and the root-directory guard.

kdeps permits paths under `~/.kdeps` and `/tmp` with no prompt, also when the workspace root is a different folder. On Windows, the permitted temp directory is the process temp directory. kdeps blocks a directory that has only the same name, and it always blocks `/`. kdeps converts a relative path to the absolute path in the working directory. Then it checks that path and gives it to the script.

Permission modes, lean mode, and harness reminders stay on [Built-in tools](/workflow/agent-loop/tools).

Each of these tools is also a workflow component. The component POSTs to `/apis/tools.kdeps.io/v1/tools/{name}/call` and passes `file_path` (`path` for `list_files`). The session's last file, `match_id`, and the colored diff stay in the agent loop.

### File operations

Always available. No environment variables required.

| Tool | Description |
|------|-------------|
| `read_file` | Read a text file with a 1-based line number on every line. PDF, DOCX, EPUB, RTF and ODT are refused with a pointer to `load_document`, which extracts their text |
| `write_file` | Write or overwrite a file with the exact bytes given (no newline translation), then read it back to verify |
| `edit_file` | Replace or append on one line: `line` or `match`, then `text` or `append` |
| `list_files` | List directory contents |
| `md5_file` | Compute a file's MD5 hash - a low-cost way to check if the content changed |
| `tail_file` | Read the last N lines of a file without loading the whole thing |

### Line numbers

`read_file` prefixes every line with its 1-based number and renders it `cat -A` style: tabs as `^I`, other control characters as `^X`, and `$` at the true line end (`  42⇥code$`). Indentation, trailing whitespace and stray control characters stay visible. `edit_file`'s `line` uses these numbers.

### Size cap

`read_file` refuses a file, and `write_file` refuses content, larger than the `file-read-limit` event (1 MiB by default). The agent loop passes the event's value to the script. A workflow component that calls the tool gets the 1 MiB default.

```yaml
# ~/.kdeps/events/file-read-limit.yaml
name: file-read-limit
on:
  bytes: 4194304  # read_file and write_file refuse anything past 4 MiB
run: reject
```

### Optional file_path

`read_file` treats `file_path` as optional: omit it and it reads the file most recently read, edited, or written this session. This helps when the model means "the file that I read last" and calls `read_file` with only `offset`/`limit`. `write_file`, `edit_file`, `md5_file`, and `tail_file` always need an explicit path.

### Colored diff

`write_file` and `edit_file` print a colored diff under the tool call: removed lines red, added green, with a few context lines. kdeps shortens large diffs (for example, a full new file). The diff shows only in the terminal. The model gets a short result.

### Jumping from search_local to read_file with match_id

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

Line edits with `edit_file` are on [Edit file](/reference/agent-loop/tool-reference#edit-file).

## Edit file

`edit_file` changes one line of an existing file. Give the line number (or words that are on the line). Then give the replacement text or the text to append. It is a [declared python tool](/reference/agent-loop/tools#declared-python-tools) in `tools/edit_file.yaml`. That YAML file contains all of its behavior. Reads, whole-file writes and `match_id` are on [File tools](/reference/agent-loop/tool-reference#file-tools).

```text
read_file (line numbers) -> edit_file {file_path, line | match, text | append} -> edited line(s), numbered
```

### Replace a line

```json
{"file_path": "/repo/app.py", "line": 12, "text": "    timeout = 30"}
```

Line 12 becomes `    timeout = 30`. `text` can have several lines. They replace the one line.

If `text` has no indentation, the tool adds the indentation of line 12 to each line. `"text": "timeout = 30"` gives the same result.

### Replace words on a line

```json
{"file_path": "/repo/app.py", "line": 12, "match": "30", "text": "60"}
```

Only the first `30` on line 12 becomes `60`. Without `line`, the tool uses the first line that contains `match`.

### Append

```json
{"file_path": "/repo/app.py", "line": 12, "append": "    retries = 3"}
```

The tool inserts `    retries = 3` after line 12. If you give no `line` and no `match`, the text goes after the end of the file.

### Result and errors

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

## Shell execution

`bash_exec`, one of the [built-in tools](/workflow/agent-loop/tools), runs a shell command and returns its output to the model.

kdeps removes HTML escapes from a command before it validates and runs the command. If the text went through an HTML step before, some models write `&amp;&amp;` for `&&`, `&quot;` for `"`, or `&lt;`/`&gt;` for `<`/`>`. kdeps does the same for `sql_query`, because an escaped comparison (`WHERE x &lt; 5`) is the same problem there.

A non-zero exit returns that output with the error. The model sees stderr and the end of stdout, not only `exit status 1`. A long `go test` log keeps the summary at the end.

Use these two related tools to manage background jobs:

| Tool | Description |
|------|-------------|
| `bash_job_list` | Show all background jobs with status (`running`/`done`/`failed`), elapsed time, and command |
| `bash_job_wait` | Block until a job completes and return its full output. Pass the job's `job_id`. |

Set `KDEPS_ALLOW_BASH=false` to disable all three `bash_*` tools.

## Memory and identity

`memory_save`, `memory_search`, `memory_delete`, and `memory_query` read and write the agent store during a turn. `identity_get` returns the configured name, email, and address. They are always registered. The rules for every tool are on [Built-in tools](/workflow/agent-loop/tools).

### Memory tools

Always available. You do not need environment variables.

`memory_save`, `memory_search`, and `memory_delete` run through the [kdeps API](/reference/api). A call with no memory store returns `memory_save: memory store is not configured`, and the same sentence for `memory_search` and `memory_delete`. The agent attaches its store before the call.

`memory_query` runs through the [kdeps API](/reference/api). An empty query returns `memory_query: query is required`. A call with no agent loop returns `memory_query: agent loop is not configured (memory_query is an agent-mode tool)`. The agent installs the loop before the call.

| Tool | Description |
|------|-------------|
| `memory_save` | Save a fact to persistent memory. Injected into every LLM call automatically. |
| `memory_search` | Search memory entries by key or value. At most 20 matches, best first (key hit, then more query words, then newer); each value cut at 500 characters. |
| `memory_delete` | Remove a memory entry by key. |
| `memory_query` | Run an expr-lang relational query over agent state: `memory` (persistent entries) and `tool_calls` (recent tool call history). Supports `filter()`, `map()`, `join()`, `union()`. String matching is infix: `.value contains "x"`, `.key startsWith "x"`, `.value matches "regex"`; `contains(...)` is a parse error. |

kdeps keeps memory for each project as markdown notes in the knowledge vault, `<vault>/projects/<project>/memory/` (see [Persistent memory](/workflow/agent-loop/memory)). Facts stay across sessions. kdeps extracts facts from each turn automatically. A `[MEMORY: key] value` line saves a fact with no call to `memory_save`.

The `memory_*` tools are how the *model* reads and writes memory during a turn.

### Identity tool

Always available. `identity_get` returns the agent's configured name, email, and address - see [Agent Identity](/reference/advanced-config#agent-identity) for how to set one. Returns "No identity configured for this agent." when unset. Never returns account credentials, even if configured. A model that can read a password can leak it in its own output.

## Data tools

`calculator`, the SQLite tools, and the local embedding store are always on. The SQLite tools use `db_path`. If `db_path` is not set, they use `KDEPS_SQL_DB_PATH`. Wolfram, remote RAG, and the rerankers register only when you set their key or URL. They are [declared HTTP tools](/reference/agent-loop/tools#declared-http-tools), and their tool asset fully defines them. Before `sql_query` text runs, kdeps removes its HTML escapes, as for [`bash_exec`](/reference/agent-loop/tool-reference#shell-execution).

Permission modes stay on [Built-in tools](/workflow/agent-loop/tools).

### Computation

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `calculator` | (none) | Evaluate an arithmetic expression: numbers, `+ - * / // % **`, parentheses and the `math` module's functions and constants (`sqrt(16)`, `log(8, 2)`, `pi`). A `python:` block in `tools/calculator.yaml`; exponents above 10000 are rejected |
| `wolfram_alpha` | `WOLFRAM_APP_ID` | Wolfram Alpha queries |

### Data and SQL

The three SQLite tools are `python:` blocks in their tool assets (Python's built-in `sqlite3`). Each tool opens the file read-only (`mode=ro`), so a query cannot write to it. First, the tool compares `db_path` with `KDEPS_WORKSPACE_ROOT`. Paths under `~/.kdeps` and `/tmp` are also permitted. `sql_ops` (named connections, any driver) stays in Go.

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `sql_list_tables` | `db_path` or `KDEPS_SQL_DB_PATH` | List the tables in a SQLite file |
| `sql_describe_table` | same | The table's `CREATE TABLE` statement |
| `sql_query` | same | Run one `SELECT` or `WITH` query; tab-separated rows, column names first |

### Embeddings and reranking

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `embedding_vectorize` | (none) | Convert text to embeddings and index it in the local embedding DB |
| `embedding_search` | (none) | Semantic search over the local embedding DB |
| `retrieve_context` | `KDEPS_RAG_BASE_URL` | Retrieve chunks from a remote RAG endpoint (only registered when the URL is set) |
| `cohere_rerank` | `COHERE_API_KEY` | Rerank results using Cohere |
| `voyageai_rerank` | `VOYAGEAI_API_KEY` | Rerank results using VoyageAI |
| `jina_rerank` | `JINA_API_KEY` | Rerank results using Jina |

## Action tools

`http_request`, `search_local`, `transcribe_audio`, `transcribe_audio_offline`, `ocr_image`, and `load_document` are always on. Zapier and Google cache register when you set their credentials (the Zapier tools are [declared HTTP tools](/reference/agent-loop/tools#declared-http-tools)). Commits the agent creates carry a co-author trailer. Permission modes stay on [Built-in tools](/workflow/agent-loop/tools).

### Git commit attribution

Commits that the agent creates have a co-author trailer with the names of kdeps and the model that wrote them. The trailer records the model that made the commit:

```text
Co-Authored-By: kdeps (deepseek/deepseek-reasoner) <noreply@kdeps.com>
```

The model name changes with the location where the model runs:

| Model | Trailer |
|-------|---------|
| Cloud provider | `kdeps (deepseek/deepseek-reasoner) <noreply@kdeps.com>` |
| Cloud provider | `kdeps (openai/gpt-4o-mini) <noreply@kdeps.com>` |
| Ollama | `kdeps (ollama/llama3.2) <noreply@kdeps.com>` |
| llamafile | `kdeps (hfuser/gemma4-2-9b llamafile) <noreply@kdeps.com>` |
| GGUF | `kdeps (hfuser/gemma4-2-9b gguf) <noreply@kdeps.com>` |

Cloud and Ollama model names start with the provider (`provider/model`). Local llamafile and GGUF names already have their HuggingFace namespace, so kdeps adds the runtime to the end. Often the same repository has both forms.

With no model configured, the trailer falls back to `Co-Authored-By: kdeps <noreply@kdeps.com>`.

A configured [identity](/reference/advanced-config#agent-identity) takes priority over all of the above: with `identity.name`/`identity.email` set, the trailer becomes `Co-Authored-By: Sales Bot <sales-bot@example.com>` and does not give the model name.

### Actions and integrations

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `zapier_list_actions` | `ZAPIER_NLA_API_KEY` | List available Zapier NLA actions |
| `zapier_run_action` | `ZAPIER_NLA_API_KEY` | Execute a Zapier NLA action |
| `google_cache_create` | `GOOGLE_API_KEY` | Create a Google AI cached content object |
| `google_cache_list` | `GOOGLE_API_KEY` | List Google AI cached content objects |
| `google_cache_delete` | `GOOGLE_API_KEY` | Delete a Google AI cached content object |

`google_cache_create`, `google_cache_delete`, and `google_cache_list` run through the [kdeps API](/reference/api). The agent registers them for the model only when you set `GOOGLE_API_KEY`. A call with the key unset returns `google_cache_create: disabled (GOOGLE_API_KEY is unset)`, and the same sentence for `google_cache_delete` and `google_cache_list`.

### Resource-backed tools

These always-on tools invoke the corresponding kdeps executor directly:

| Tool | Description |
|------|-------------|
| `http_request` | Make an HTTP request (GET/POST/PUT/DELETE/PATCH) |
| `search_local` | Search files in a folder (`path`), or, with no `path`, the working directory. Dot-directories, vendor/build folders (`node_modules`, `vendor`, `target`, caches), and files over 1 MiB are skipped. Key and credential files are never returned |
| `transcribe_audio` | Transcribe an audio file through a Whisper API (OpenAI, Groq, or a local HTTP server); a declared HTTP tool |
| `transcribe_audio_offline` | Transcribe offline with whisper.cpp (`whisper-cli`), no API key; a declared exec tool |
| `ocr_image` | Extract text from an image via tesseract (local, no API key); a [declared exec tool](/reference/agent-loop/tools#declared-exec-tools) |
| `load_document` | Load and extract text from a document |

## Harness reminders

`remind: true` on a harness section puts its text onto every prompt and every tool result. Permission modes stay on [Built-in tools](/workflow/agent-loop/tools).

### Every prompt and every tool result

Each harness section has a normal place. A preamble section goes into the system prompt. A standalone entry operates only at its call site. Examples are a nudge, a praise line, and the compaction, judge, and refine prompts. `remind: true` overrides this.

The raw text of the section then goes onto each prompt, from the first turn. It also goes onto each tool result, also when its trigger does not occur.

```yaml
# ~/.kdeps/harness/tools-reminder.yaml (or harness/ in the workflow project)
name: tools-reminder
kind: standalone
remind: true        # add this text to every prompt and every tool result
body: >-
  Every capability you have is a kdeps tool. Call it.
```

Use it for a model that needs guidance more often than its normal place gives: a sandbox habit, a recurring unwanted pattern, a house style. Each reminder adds its tokens to every turn and every tool result, so keep the text short.

- Reminders are independent of `disabled:`. A disabled section can still be a reminder.
- An override file replaces the shipped entry of the same name, so copy the shipped `body:` when you only want to add `remind: true`.
