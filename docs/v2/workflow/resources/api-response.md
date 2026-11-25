# API response resource

`apiResponse:` builds the HTTP response that a workflow sends back to the caller. It is the last resource in the chain: the one that [`targetActionId`](/reference/glossary#targetactionid) in `workflow.yaml` names.

```text
request -> resources (requires: order) -> apiResponse -> HTTP response
```

```yaml
# resources/respond.yaml
actionId: respond
flow:
  requires: [answer]                    # runs after the answer resource
apiResponse:
  success: true
  response:
    answer: get('answer').message.content   # the chat's reply text
  headers:
    Content-Type: application/json
```

## Read next

- [API response resource reference](/reference/resources/api-response) - every key, option and detail
- [Error handling](/workflow/error-handling) - what the caller gets when a resource fails
