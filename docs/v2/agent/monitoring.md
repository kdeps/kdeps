# Tool execution monitoring

The [agent loop REPL](/agent/) shows every running [built-in tool](/agent/tools)'s progress and kills one that stalls.

When a tool runs, a live monitor line refreshes each second: `⠴ bash_exec running (12m34s) · <latest output line>`. When the tool finishes, the usual `... done (elapsed)` summary replaces it.

The line starts with what the tool acts on, from its arguments: the URL for `web_scraper`/`http_request`, the query for `web_search`/`sql_query`, the path for `search_local`, and so on (`⠴ web_scraper running (3s) · https://example.com`). Streaming tools such as `bash_exec` then show their latest output line.

The same status line covers `! <cmd>` / `!! <cmd>` shell commands and `@file` ref expansion: when a bang command gives no output, the line shows `⠴ ! make lint running (57s)`. Real output erases the status line first, so the two do not mix.

kdeps measures a hang by the time with no output, not by the total time. A build that continues to print does not cause a hang. After 2 minutes with no output, the line shows a warning (`no output for 3m20s`). After the stall timeout, kdeps stops the tool.

The default stall timeout is 10 minutes with no output. To change it, use `/model tool set stall-timeout 5m`. `0` disables it. The model gets a structured error about the hang. It can then try again with a narrower or more verbose command, or run the command in the background.

When a tool stalls, the default is auto-increase. kdeps adds the increment (default 5m) to the stall timeout and shows a message (`[Auto-stall allocation: stall timeout increased by 5m. New timeout: 15m.]`). Then a command that runs for a long time with no output continues without a prompt. This is on by default.

To get an interactive prompt, turn off auto-increase in the configuration. Then, when a tool stalls, the REPL shows `(i)ncrease` / `(k)ill`.

Tools marked "cached" in Web and search memoize successful results for the agent process's lifetime. A repeat query or URL returns instantly. Failed and empty lookups are not cached. `wolfram_alpha` caches the same way ([Data tools](/agent/tools-data)).

## See also

- [Built-in tools](/agent/tools) - the full tool catalog
- [Shell execution](/agent/shell) - Ctrl+C/Ctrl+Z, background jobs, rtk
