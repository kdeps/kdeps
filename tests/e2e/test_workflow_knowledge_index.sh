#!/bin/bash
# Copyright 2026 Kdeps, KvK 94834768
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# E2E: a workflow's knowledge.index folders become search folders of the
# workflow's project and are indexed when kdeps run starts it.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow knowledge.index..."

KI_DIR=$(mktemp -d)
trap 'rm -rf "$KI_DIR"' EXIT
mkdir -p "$KI_DIR/home" "$KI_DIR/wf/resources" "$KI_DIR/wf/docs"
echo "# billing runs monthly" > "$KI_DIR/wf/docs/billing.md"
cat > "$KI_DIR/wf/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: knowledge-index-e2e
  version: "1.0.0"
  targetActionId: done
knowledge:
  index: [./docs]
YAML
cat > "$KI_DIR/wf/resources/done.yaml" <<'YAML'
actionId: done
name: Done
apiResponse:
  success: true
  response:
    ok: true
YAML

ki() {
    (cd "$KI_DIR/wf" && HOME="$KI_DIR/home" USERPROFILE="$KI_DIR/home" KDEPS_CONFIG_PATH="$KI_DIR/home/.kdeps/config.yaml" \
        timeout 120 "$KDEPS_BIN" "$@" 2>&1 </dev/null)
}

if ki validate workflow.yaml > /dev/null; then
    test_passed "knowledge.index - validates"
else
    test_failed "knowledge.index - validates" "$(ki validate workflow.yaml | tail -3)"
fi

ki run workflow.yaml > /dev/null
OUT=$(ki knowledge folders)
if output_grep "search .*docs" "$OUT" && output_grep_fixed "index.db" "$OUT"; then
    test_passed "knowledge.index - kdeps run records and indexes the folder"
else
    test_failed "knowledge.index - kdeps run records and indexes the folder" "$(printf '%s' "$OUT" | tail -6)"
fi

rm -rf "$KI_DIR"
trap - EXIT
