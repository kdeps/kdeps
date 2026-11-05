# Tool execution monitoring

The [agent loop REPL](/agent/) shows every running [built-in tool](/agent/tools)'s progress and kills one that stalls.

While a tool runs, a live monitor line refreshes every second: `⠴ bash_exec running (12m34s) · <latest output line>`. When the tool finishes, the usual `... done (elapsed)` summary replaces it.

The line starts with what the tool acts on, from its arguments: the URL for `web_scraper`/`http_request`, the query for `web_search`/`sql_query`, the path for `search_local`, and so on (`⠴ web_scraper running (3s) · https://example.com`). Streaming tools such as `bash_exec` then show their latest output line.

The same status line covers `! <cmd>` / `!! <cmd>` shell commands and `@file` ref expansion: while a bang command is silent the line shows `⠴ ! make lint running (57s)`, and any real output erases the status line first so the two never collide.

Hangs are measured by silence, not wall-clock time; a build that keeps printing never trips it. After 2 minutes without output the line warns (`no output for 3m20s`). After the stall timeout (default 10 minutes of silence; `/model tool set stall-timeout 5m`, `0` disables) the tool is killed. The model gets a structured error explaining the hang, so it can retry with a narrower or more verbose command, or run it in the background.

When a tool stalls, the default is auto-increase: the stall timeout is bumped by the increment (default 5m) and the bump is announced (`[Auto-stall allocation: stall timeout increased by 5m. New timeout: 15m.]`), so a long silent-but-alive command keeps running without a prompt. This is on by default.

To be prompted interactively instead, turn auto-increase off in config; the REPL then offers `(i)ncrease` / `(k)ill` when a tool stalls.

Tools marked "cached" in Web and search memoize successful results for the agent process's lifetime; a repeat query or URL returns instantly. Failed and empty lookups are not cached. `wolfram_alpha` caches the same way ([Data tools](/agent/tools-data)).

## See also

- [Built-in tools](/agent/tools) - the full tool catalog
- [Shell execution](/agent/shell) - Ctrl+C/Ctrl+Z, background jobs, rtk
