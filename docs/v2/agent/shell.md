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

## Token savings with rtk (optional)

[rtk](https://github.com/rtk-ai/rtk) is a CLI proxy that compresses command output before it reaches the LLM. `git status` costs ~300 tokens; `rtk git status` costs ~60 for the same information. If you install rtk, `bash_exec` uses it automatically. You do not configure it.

```text
LLM calls bash_exec("go test ./...")
  -> kdeps asks: rtk rewrite "go test ./..."
  -> rtk answers: rtk go test ./...
  -> kdeps runs the rewritten command
  -> LLM sees filtered output (up to 90% fewer tokens)
```

Install it with `brew install rtk`, or skip it - kdeps runs your commands unchanged when rtk is absent.

| Env var | Effect |
|---------|--------|
| _(none)_ | Auto-detect. rtk is used when it is on `PATH` and passes verification. |
| `KDEPS_RTK=off` | Never use rtk, even if installed. |
| `RTK_DISABLED=1` | Also honored. rtk's own escape hatch, so one variable turns it off everywhere. |

What this does not change:

- **Your commands still run.** If rtk is missing, too old, wedged, or has no compression for a command, kdeps runs the original. The same applies when rtk's rewrite does not invoke `rtk`; kdeps runs only a rewrite it can identify as an rtk command. rtk can never block execution.
- **Permissions do not change.** kdeps controls shell commands itself. Here, rtk is only a compressor. kdeps ignores the permission decisions of rtk, so you do not get two permission checks.
- **A workflow `exec` resource keeps raw output,** because pipelines parse it downstream. `bash_exec` and the shipped `bash_exec` component both use rtk.

::: tip Verifying the right rtk
An unrelated crate on crates.io is also named `rtk`. kdeps verifies the binary by its behavior, not by its name. For this reason, kdeps ignores a different binary with this name on your `PATH`. Check yours with `rtk gain`: it works on the real one.
:::

## See also

- [Built-in tools](/agent/tools) - the full tool catalog
- [Tool execution monitoring](/agent/monitoring) - live status lines and stall detection
