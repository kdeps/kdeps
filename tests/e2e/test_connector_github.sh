#!/usr/bin/env bash
# E2E: the github_get and github_post connector components. No network: the
# run has no token, so the call stops before it reaches api.github.com.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing the GitHub connector..."

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

rm -rf "$GH_DIR"
echo ""
