#!/usr/bin/env bash
# E2E: the connector components (GitHub, Slack, Notion, Airtable, Stripe, Jira). No network: the runs have
# no token, so each call stops before it reaches the service.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing the connectors..."

GH_DIR=$(mktemp -d)
mkdir -p "$GH_DIR/home" "$GH_DIR/proj/resources"

cat > "$GH_DIR/proj/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: github-connector-e2e
  version: "1.0.0"
  targetActionId: response
YAML
cat > "$GH_DIR/proj/resources/issues.yaml" <<'YAML'
actionId: issues
name: Open issues
component:
  name: github_get
  with:
    path: repos/kdeps/kdeps/issues?state=open
YAML
cat > "$GH_DIR/proj/resources/comment.yaml" <<'YAML'
actionId: comment
name: Comment
flow:
  requires: [issues]
component:
  name: github_post
  with:
    path: repos/kdeps/kdeps/issues/1/comments
    body:
      body: "seen by kdeps"
YAML
cat > "$GH_DIR/proj/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [comment]
apiResponse:
  success: true
  response:
    issues: "{{ output('issues') }}"
YAML

ghrun() {
    (cd "$GH_DIR/proj" && HOME="$GH_DIR/home" USERPROFILE="$(to_native_path "$GH_DIR/home")" \
        GITHUB_TOKEN="" GH_TOKEN="" KDEPS_VAULT_PASSWORD="" "$KDEPS_BIN" "$@")
}

OUT=$(ghrun validate workflow.yaml 2>&1)
if [ $? -eq 0 ]; then
    test_passed "github connector - a workflow with github_get and github_post validates"
else
    test_failed "github connector - validate" "$OUT"
fi

OUT=$(ghrun run workflow.yaml </dev/null 2>&1)
if output_grep "github_get: required environment variables are not set: set GITHUB_TOKEN or GH_TOKEN or the vault secret github_token" "$OUT"; then
    test_passed "github connector - no token names GITHUB_TOKEN, GH_TOKEN and the vault secret"
else
    test_failed "github connector - no token" "$OUT"
fi

mkdir -p "$GH_DIR/slack/resources"
cp "$GH_DIR/proj/workflow.yaml" "$GH_DIR/slack/workflow.yaml"
cat > "$GH_DIR/slack/resources/history.yaml" <<'YAML'
actionId: history
name: Channel history
component:
  name: slack_get
  with:
    path: conversations.history?channel=C0123&limit=5
YAML
cat > "$GH_DIR/slack/resources/notify.yaml" <<'YAML'
actionId: notify
name: Notify
flow:
  requires: [history]
component:
  name: slack_post
  with:
    path: chat.postMessage
    body:
      channel: C0123
      text: "seen by kdeps"
YAML
cat > "$GH_DIR/slack/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [notify]
apiResponse:
  success: true
  response:
    sent: "{{ output('notify') }}"
YAML

slackrun() {
    (cd "$GH_DIR/slack" && HOME="$GH_DIR/home" USERPROFILE="$(to_native_path "$GH_DIR/home")" \
        SLACK_BOT_TOKEN="" SLACK_TOKEN="" KDEPS_VAULT_PASSWORD="" "$KDEPS_BIN" "$@")
}

OUT=$(slackrun validate workflow.yaml 2>&1)
if [ $? -eq 0 ]; then
    test_passed "slack connector - a workflow with slack_get and slack_post validates"
else
    test_failed "slack connector - validate" "$OUT"
fi

OUT=$(slackrun run workflow.yaml </dev/null 2>&1)
if output_grep "slack_get: required environment variables are not set: set SLACK_BOT_TOKEN or SLACK_TOKEN or the vault secret slack_token" "$OUT"; then
    test_passed "slack connector - no token names SLACK_BOT_TOKEN, SLACK_TOKEN and the vault secret"
else
    test_failed "slack connector - no token" "$OUT"
fi

mkdir -p "$GH_DIR/notion/resources"
cp "$GH_DIR/proj/workflow.yaml" "$GH_DIR/notion/workflow.yaml"
cat > "$GH_DIR/notion/resources/find.yaml" <<'YAML'
actionId: find
name: Search
component:
  name: notion_post
  with:
    path: search
    body:
      query: roadmap
YAML
cat > "$GH_DIR/notion/resources/page.yaml" <<'YAML'
actionId: page
name: Read the page
flow:
  requires: [find]
component:
  name: notion_get
  with:
    path: pages/PAGE_ID
YAML
cat > "$GH_DIR/notion/resources/archive.yaml" <<'YAML'
actionId: archive
name: Archive the page
flow:
  requires: [page]
component:
  name: notion_patch
  with:
    path: pages/PAGE_ID
    body:
      archived: true
YAML
cat > "$GH_DIR/notion/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [archive]
apiResponse:
  success: true
  response:
    page: "{{ output('archive') }}"
YAML

notionrun() {
    (cd "$GH_DIR/notion" && HOME="$GH_DIR/home" USERPROFILE="$(to_native_path "$GH_DIR/home")" \
        NOTION_TOKEN="" NOTION_API_KEY="" KDEPS_VAULT_PASSWORD="" "$KDEPS_BIN" "$@")
}

OUT=$(notionrun validate workflow.yaml 2>&1)
if [ $? -eq 0 ]; then
    test_passed "notion connector - a workflow with notion_post, notion_get and notion_patch validates"
else
    test_failed "notion connector - validate" "$OUT"
fi

OUT=$(notionrun run workflow.yaml </dev/null 2>&1)
if output_grep "notion_post: required environment variables are not set: set NOTION_TOKEN or NOTION_API_KEY or the vault secret notion_token" "$OUT"; then
    test_passed "notion connector - no token names NOTION_TOKEN, NOTION_API_KEY and the vault secret"
else
    test_failed "notion connector - no token" "$OUT"
fi

mkdir -p "$GH_DIR/batch/resources"
cp "$GH_DIR/proj/workflow.yaml" "$GH_DIR/batch/workflow.yaml"
cat > "$GH_DIR/batch/resources/rows.yaml" <<'YAML'
actionId: rows
name: Airtable rows
component:
  name: airtable_get
  with:
    path: appX/Tasks?maxRecords=5
YAML
cat > "$GH_DIR/batch/resources/customer.yaml" <<'YAML'
actionId: customer
name: Stripe customer
flow:
  requires: [rows]
component:
  name: stripe_post
  with:
    path: customers
    body:
      email: a@example.com
      metadata:
        plan: pro
YAML
cat > "$GH_DIR/batch/resources/issue.yaml" <<'YAML'
actionId: issue
name: Jira issue
flow:
  requires: [customer]
component:
  name: jira_put
  with:
    path: issue/KEY-1
    body:
      fields:
        summary: Updated by kdeps
YAML
cat > "$GH_DIR/batch/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [issue]
apiResponse:
  success: true
  response:
    rows: "{{ output('rows') }}"
YAML

batchrun() {
    (cd "$GH_DIR/batch" && HOME="$GH_DIR/home" USERPROFILE="$(to_native_path "$GH_DIR/home")" \
        AIRTABLE_TOKEN="" AIRTABLE_API_KEY="" STRIPE_SECRET_KEY="" STRIPE_API_KEY="" \
        JIRA_URL="" JIRA_EMAIL="" JIRA_API_TOKEN="" KDEPS_VAULT_PASSWORD="" "$KDEPS_BIN" "$@")
}

OUT=$(batchrun validate workflow.yaml 2>&1)
if [ $? -eq 0 ]; then
    test_passed "airtable, stripe and jira connectors - a workflow with all three validates"
else
    test_failed "airtable, stripe and jira connectors - validate" "$OUT"
fi

OUT=$(batchrun run workflow.yaml </dev/null 2>&1)
if output_grep "airtable_get: required environment variables are not set: set AIRTABLE_TOKEN or AIRTABLE_API_KEY or the vault secret airtable_token" "$OUT"; then
    test_passed "airtable connector - no token names AIRTABLE_TOKEN, AIRTABLE_API_KEY and the vault secret"
else
    test_failed "airtable connector - no token" "$OUT"
fi

rm -rf "$GH_DIR"
echo ""
