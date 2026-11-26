# Connectors

A connector is a component for one web service. It fixes the host and the auth, so a step names only the API path. Each connector reads its token from the environment or from the [vault](/workflow/secrets).

```text
component: github_get -> token (env or vault) -> https://api.github.com/PATH -> JSON
```

```yaml
# resources/issues.yaml  (kdeps secret set github_token, one time)
actionId: issues
component:
  name: github_get                    # GET; github_post creates issues and comments
                                      # slack_* and notion_* work the same way
  with:
    path: repos/OWNER/REPO/issues?state=open
```

## Read next

- [Connectors reference](/reference/resources/connectors) - every connector, its token and its tool tier
- [Secrets](/workflow/secrets) - store a token in the encrypted vault
- [HTTP client](/workflow/resources/http-client) - any other HTTP API
