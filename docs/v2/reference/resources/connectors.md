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
| `slack_get` | GET | `SLACK_BOT_TOKEN`, `SLACK_TOKEN`, vault `slack_token` | T0 (read-only) | channels, messages, threads, users |
| `slack_post` | POST | `SLACK_BOT_TOKEN`, `SLACK_TOKEN`, vault `slack_token` | T1 (workspace-write) | send, update and delete messages, reactions |
| `notion_get` | GET | `NOTION_TOKEN`, `NOTION_API_KEY`, vault `notion_token` | T0 (read-only) | pages, properties, blocks, users |
| `notion_post` | POST | `NOTION_TOKEN`, `NOTION_API_KEY`, vault `notion_token` | T1 (workspace-write) | search, database queries, new pages |
| `notion_patch` | PATCH | `NOTION_TOKEN`, `NOTION_API_KEY`, vault `notion_token` | T1 (workspace-write) | page properties, new blocks, archive |

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

## Slack

`path` is the Web API method after `https://slack.com/api/`. A `slack_get` path holds the arguments as a query string; `slack_post` sends them as a JSON body. Use a bot token (`xoxb-...`) with the scopes the method needs.

<div v-pre>

```yaml
# resources/history.yaml
actionId: history
name: Last messages
component:
  name: slack_get
  with:
    path: conversations.history?channel=C0123&limit=20
```

```yaml
# resources/notify.yaml
actionId: notify
name: Post the summary
component:
  name: slack_post
  with:
    path: chat.postMessage
    body:                              # sent as JSON
      channel: C0123
      text: "{{ output('summary') }}"
```

</div>

The output is the parsed JSON that Slack returns. Slack answers a failed call with status 200, `"ok": false` and an `error` text (`invalid_auth`, `channel_not_found`), so the step does not fail. Test `output('notify').ok` in the next step.

## Notion

`path` is the part of the URL after `https://api.notion.com/v1/`. Use an internal integration token, and share each page or database with the integration in Notion. Each request sends `Notion-Version: 2022-06-28`.

<div v-pre>

```yaml
# resources/tasks.yaml
actionId: tasks
name: Open tasks
component:
  name: notion_post                    # search and database queries are POST in Notion
  with:
    path: databases/DATABASE_ID/query
    body:
      filter: { property: Status, status: { equals: Open } }
```

```yaml
# resources/done.yaml
actionId: done
name: Close the task
component:
  name: notion_patch
  with:
    path: "pages/{{ get('page_id') }}"
    body:
      properties:
        Status: { status: { name: Done } }
```

</div>

The output is the parsed JSON that Notion returns. A status of 400 or higher fails the step with the Notion error message.

## Add a connector

A connector is two YAML files and no Go code:

1. `assets/tools/<name>.yaml` - the `http:` block. List the token sources under `env:`, for example `TOKEN: [SERVICE_TOKEN, "secret:service_token"]`.
2. `assets/components/<name>/component.yaml` - calls `http://kdeps.api/apis/tools.kdeps.io/v1/tools/<name>/call`.

Give the tool a tier in `assets/policy/permission.yaml`: `T0` for reads, `T1` for writes.
