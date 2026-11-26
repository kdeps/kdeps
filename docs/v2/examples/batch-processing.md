# Process a batch of items in one request

Build an HTTP API that takes a list of items in one request. The API gets data
for each item, transforms each result, and returns an aggregated summary. It
uses `items:` iteration: a resource runs one time for each element in a list.

The fetch resource runs one time for each item:

<div v-pre>

```yaml
# resources/01-fetch.yaml
actionId: fetch
name: Fetch repo data
flow:
  items:
    - "{{ input.items }}"           # the "items" array from the request body
component:
  name: http_request
  with:
    method: GET
    url: "https://api.github.com/repos/{{ item.org }}/{{ item.repo }}"
    headers:
      Accept: application/vnd.github.v3+json
    timeout: 10s
```

</div>

## Read next

- [Full walkthrough](/reference/examples/batch-processing) - every step, file and command
- [Items iteration](/workflow/items) - `item.prev()`, `item.next()`, skipping
- [While-loop iteration](/workflow/loop) - iteration with no fixed limit, when you do not know the count
