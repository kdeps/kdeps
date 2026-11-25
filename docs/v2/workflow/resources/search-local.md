# search_local component

The `search_local` component searches a local directory by content keyword and/or filename glob and returns `{results, count}`.

<div v-pre>

```yaml
# resources/search.yaml
actionId: search
name: Search
component:
  name: search_local              # from the components asset set; a components/search_local/ of your own wins
  with:
    path: "{{ get('path', '/data') }}"   # directory to search; empty searches the working project's folders
    query: "{{ get('query') }}"          # keyword in file contents; query or glob is required
```

</div>

## Read next

- [search_local component reference](/reference/resources/search-local) - every key, option and detail
- [Search resources](/workflow/resources/search) - overview and `web_search`
- [Knowledge](/workflow/agent-loop/knowledge) - the vault graph `graphBoost` reads when `path` is inside the vault
