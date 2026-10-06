# Web and search

These tools fetch the web during an agent turn. DuckDuckGo, Wikipedia, and the page scraper need no key. SerpAPI, Exa, and Perplexity register only when their env var is set. Permission modes stay on [Built-in tools](/agent/tools).

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `web_search` | (none - uses DuckDuckGo) | Search the web (30s timeout, cached) |
| `wikipedia` | (none) | Fetch a Wikipedia article (30s timeout, cached) |
| `web_scraper` | (none) | Fetch and extract text from any URL (60s timeout, cached) |
| `serpapi_search` | `SERPAPI_API_KEY` | Google search via SerpAPI (30s timeout, cached) |
| `exa_search` | `EXA_API_KEY` or `METAPHOR_API_KEY` | Neural search via Exa (cached) |
| `perplexity_search` | `PERPLEXITY_API_KEY` | Search via Perplexity (30s timeout, cached) |

Web and search tools carry a hard timeout so a hung remote endpoint cannot stall the turn. Ctrl+C during any tool call cancels the in-flight request immediately and skips the round's remaining tools. Tools marked "cached" memoize successful results for the process lifetime; failed/empty lookups are retried.

While any tool runs, the REPL shows a live status line and detects hangs via a stall timeout - see [Tool Execution Monitoring](/agent/monitoring) for the full mechanics.
