# Web scraper component

The `web_scraper` component fetches one URL and returns its readable text: title, headers, body and links, without navigation and script noise.

```yaml
# resources/scrape.yaml
actionId: scrape
name: Read Page
component:
  name: web_scraper               # from the components asset set; a components/web_scraper/ of your own wins
  with:
    url: "https://example.com/article"   # required, full URL with https://
```

## Read next

- [Web scraper component reference](/reference/resources/web-scraper) - every key, option and detail
- [Search resources](/workflow/resources/search) - find URLs with `web_search` first
- [HTTP client](/workflow/resources/http-client) - JSON APIs, auth, retry, cache
