# Shell execution

`bash_exec`, one of the [built-in tools](/agent/tools), runs a shell command and returns its output to the model.

kdeps removes HTML escapes from a command before it validates and runs the command. If the text went through an HTML step before, some models write `&amp;&amp;` for `&&`, `&quot;` for `"`, or `&lt;`/`&gt;` for `<`/`>`. kdeps does the same for `sql_query`, because an escaped comparison (`WHERE x &lt; 5`) is the same problem there.

A non-zero exit returns that output with the error. The model sees stderr and the end of stdout, not only `exit status 1`. A long `go test` log keeps the summary at the end.

Use these two related tools to manage background jobs:

| Tool | Description |
|------|-------------|
| `bash_job_list` | Show all background jobs with status (`running`/`done`/`failed`), elapsed time, and command |
| `bash_job_wait` | Block until a job completes and return its full output. Pass the job's `job_id`. |

Set `KDEPS_ALLOW_BASH=false` to disable all three `bash_*` tools.

## See also

- [Built-in tools](/agent/tools) - the full tool catalog
