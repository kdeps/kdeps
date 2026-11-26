# Connectors reference

A connector is a kdeps component for one web service. It calls a tool asset whose `http:` block fixes the host, the headers and the auth. A workflow step and the agent loop use the same tool.

```d2
direction: right
step: "component: github_get" {shape: oval}
api: "kdeps API\ntools/github_get"
env: "GITHUB_TOKEN, GH_TOKEN"
vault: "vault secret github_token"
svc: "https://api.github.com/PATH"
out: "JSON body" {shape: oval}
step -> api
env -> api: "1st"
vault -> api: "2nd"
api -> svc
svc -> out
```

## The token

A connector reads its token from the first source that has a value: the environment variables, then the vault secret. Store the token one time:

```bash
kdeps secret set github_token     # project vault when the folder holds workflow.yaml
```

With no token, the call fails and names every source it tried:

```text
github_get: required environment variables are not set: set GITHUB_TOKEN or GH_TOKEN or the vault secret github_token
```

In the agent loop, a connector tool with no token is not offered to the model.

## Connectors

| Component | Method | Token sources | Tool tier | Use it for |
|---|---|---|---|---|
| `github_get` | GET | `GITHUB_TOKEN`, `GH_TOKEN`, vault `github_token` | T0 (read-only) | repositories, issues, pull requests, commits, files, search |
| `github_post` | POST | `GITHUB_TOKEN`, `GH_TOKEN`, vault `github_token` | T1 (workspace-write) | new issues, comments, pull requests, labels |

The tool tier comes from `assets/policy/permission.yaml`. A `T1` tool needs the `workspace-write` permission mode or higher.

## GitHub

`path` is the part of the URL after `https://api.github.com/`, with its query string. The host is fixed: `path` cannot send the token to another host.

<div v-pre>

```yaml
# resources/issues.yaml
actionId: issues
name: Open issues
component:
  name: github_get
  with:
    path: "repos/{{ get('repo') }}/issues?state=open&per_page=20"
```

```yaml
# resources/report.yaml
actionId: report
name: Report a bug
component:
  name: github_post
  with:
    path: repos/OWNER/REPO/issues
    body:                              # sent as JSON
      title: "{{ get('title') }}"
      body: "{{ get('details') }}"
      labels: [bug]
```

</div>

The output is the parsed JSON that GitHub returns. A status of 400 or higher fails the step with the GitHub error message.

Each request sends `Authorization: Bearer <token>`, `Accept: application/vnd.github+json`, `X-GitHub-Api-Version: 2022-11-28` and `User-Agent: kdeps`.

## Add a connector

A connector is two YAML files and no Go code:

1. `assets/tools/<name>.yaml` - the `http:` block. List the token sources under `env:`, for example `TOKEN: [SERVICE_TOKEN, "secret:service_token"]`.
2. `assets/components/<name>/component.yaml` - calls `http://kdeps.api/apis/tools.kdeps.io/v1/tools/<name>/call`.

Give the tool a tier in `assets/policy/permission.yaml`: `T0` for reads, `T1` for writes.
