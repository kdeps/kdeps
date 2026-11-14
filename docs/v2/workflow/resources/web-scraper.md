# Web scraper component

The `web_scraper` component fetches one URL and returns its readable text: title, headers, body and links, without navigation and script noise.

It comes from the `components` asset set. Call it with a `component:` resource. You do not install it. The `web_scraper` tool gets the page. This tool is a standard-library `python:` block in `tools/web_scraper.yaml`, and the kdeps API serves it. It gets the page one time and keeps the title, meta description, headings, paragraphs and links.

## Where it runs

Both [workflow mode](/workflow/) and [agent mode](/agent/). In agent mode, the same tool is the `web_scraper` built-in, with the aliases `fetch`, `web_fetch`, `fetch_url`, and `read_url`. The component and the agent call the same route.

## Basic usage

```text
component: web_scraper -> POST http://kdeps.api/apis/tools.kdeps.io/v1/tools/web_scraper/call
                       -> fetch the page, drop scripts and navigation
                       -> the readable text
```

```yaml
# resources/scrape.yaml
actionId: scrape
name: Read Page
component:
  name: web_scraper               # from the components asset set; a components/web_scraper/ of your own wins
  with:
    url: "https://example.com/article"   # required, full URL with https://
```

```yaml
# resources/respond.yaml
actionId: respond
flow:
  requires: [scrape]
apiResponse:
  success: true
  response:
    text: "{{ output('scrape') }}"
```

## Inputs

| Input | Default | Description |
|---|---|---|
| `url` | (required) | Full URL, including `https://`. Authenticated and private URLs fail |

## Output

The page's readable text, as a string. A page that only renders with JavaScript comes back nearly empty - fetch its JSON API with `http_request` instead.

## See also

- [Search resources](/workflow/resources/search) - find URLs with `web_search` first
- [HTTP client](/workflow/resources/http-client) - JSON APIs, auth, retry, cache
- [Resources overview](/workflow/resources) - all resource types
