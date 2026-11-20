# Shell execution

`bash_exec`, one of the [built-in tools](/agent/tools), runs any shell command and streams output to the terminal. Two keyboard shortcuts change its behavior mid-run.

kdeps removes HTML escapes from a command before it validates and runs the command. If the text went through an HTML step before, some models write `&amp;&amp;` for `&&`, `&quot;` for `"`, or `&lt;`/`&gt;` for `<`/`>`. kdeps does the same for `sql_query`, because an escaped comparison (`WHERE x &lt; 5`) is the same problem there.

A non-zero exit returns that output with the error. The model sees stderr and the end of stdout, not only `exit status 1`. A long `go test` log keeps the summary at the end.

| Key | Effect |
|-----|--------|
| `Ctrl+C` | Cancel the running tool. Partial output is returned to the LLM as a result so it can decide what to do next. Works for any built-in tool, not only `bash_exec`. |
| `Ctrl+Z` | Detach the process as a background job. `bash_exec` immediately returns `{"status":"backgrounded","job_id":N}` to the LLM. |

When a tool runs, kdeps reads Ctrl+C directly from the terminal. For this reason, Ctrl+C also cancels tools that run for a long time (for example, a slow `search_local` or `web_scraper`). The REPL does not need the terminal to send a signal.

`Ctrl+Z` at the REPL prompt (no tool running) suspends kdeps normally (`fg` to resume).

Use these two related tools to manage background jobs:

| Tool | Description |
|------|-------------|
| `bash_job_list` | Show all background jobs with status (`running`/`done`/`failed`), elapsed time, and command |
| `bash_job_wait` | Block until a job completes and return its full output. Pass `job_id` from the backgrounded result. |

Set `KDEPS_ALLOW_BASH=false` to disable all three `bash_*` tools.

## See also

- [Built-in tools](/agent/tools) - the full tool catalog
- [Tool execution monitoring](/agent/monitoring) - live status lines and stall detection
