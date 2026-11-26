# Run extra resources inline

Attach `component:` and `chat:` actions directly to the `before:` and `after:`
blocks of one resource. You do not make a separate file for each action. The inline actions run as part of the main resource.

Try it:

```bash
curl -X POST http://localhost:16396/api/v1/process \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"data": "hey can you send me that file"}'
```

## Read next

- [Full walkthrough](/reference/examples/inline-resources) - every step, file and command
- [Inline resources](/workflow/inline-resources) - all supported types, ordering
- [Error handling (onError)](/workflow/error-handling) - fallback for the whole resource
