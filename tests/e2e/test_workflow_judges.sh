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

# E2E: a workflow chat with judges:. The mock LLM writes a DRAFT, both judges
# reject it, the chat runs again with their feedback, and the response
# carries the revision. A grader-only judge settles without a model call.
# judgePreset: design-review runs that preset's judges the same way.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat judges..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow judges - python3 not available"
    return 0 2>/dev/null || exit 0
fi

API_PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT

cat > "$TEST_DIR/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: judges-e2e
  version: "1.0.0"
  targetActionId: response
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${API_PORT}
    routes:
      - path: /notes
        methods: [GET]
YAML

cat > "$TEST_DIR/resources/notes.yaml" <<'YAML'
actionId: notes
name: Release notes
chat:
  model:
    name: fake
  context:
    prompt: "JUDGED: write the release notes"
  guardrails:
    judges:
      - name: complete
        criteria: Every change is listed.
      - name: final
        criteria: The notes are final, not a draft.
YAML

cat > "$TEST_DIR/resources/graded.yaml" <<'YAML'
actionId: graded
name: Graded notes
chat:
  model:
    name: fake
  context:
    prompt: "JUDGED: write the short notes"
  guardrails:
    judges:
      - name: no-draft
        graders:
          - kind: not_contains
            pattern: DRAFT
YAML

cat > "$TEST_DIR/resources/preset.yaml" <<'YAML'
actionId: preset
name: Preset-reviewed design
chat:
  model:
    name: fake
  context:
    prompt: "JUDGED: write the design"
  guardrails:
    judgePreset: design-review
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [notes, graded, preset]
apiResponse:
  success: true
  response:
    notes: "{{ output('notes') }}"
    graded: "{{ output('graded') }}"
    preset: "{{ output('preset') }}"
YAML

if "$KDEPS_BIN" validate "$TEST_DIR/workflow.yaml" &>/dev/null; then
    test_passed "Workflow judges - workflow with judges validates"
else
    test_failed "Workflow judges - workflow with judges validates" "$("$KDEPS_BIN" validate "$TEST_DIR/workflow.yaml" 2>&1 | tail -5)"
fi

cat > "$TEST_DIR/resources/bad.yaml" <<'YAML'
actionId: bad
name: Bad judge
chat:
  model:
    name: fake
  context:
    prompt: "x"
  guardrails:
    judges:
      - name: lonely
        allowedTools: [write_file]
YAML
BAD_OUT=$("$KDEPS_BIN" validate "$TEST_DIR/workflow.yaml" 2>&1)
if echo "$BAD_OUT" | grep -q 'chat.judges\[0\]: judge "lonely" needs criteria or graders'; then
    test_passed "Workflow judges - a judge without criteria or graders is rejected"
else
    test_failed "Workflow judges - a judge without criteria or graders is rejected" "$(echo "$BAD_OUT" | tail -5)"
fi
rm -f "$TEST_DIR/resources/bad.yaml"

cat > "$TEST_DIR/resources/both.yaml" <<'YAML'
actionId: both
name: Judges and a preset
chat:
  model:
    name: fake
  context:
    prompt: "x"
  guardrails:
    judgePreset: design-review
    judges:
      - name: extra
        criteria: c
YAML
BOTH_OUT=$("$KDEPS_BIN" validate "$TEST_DIR/workflow.yaml" 2>&1)
if echo "$BOTH_OUT" | grep -q 'set judges or judgePreset, not both'; then
    test_passed "Workflow judges - judges and judgePreset together are rejected"
else
    test_failed "Workflow judges - judges and judgePreset together are rejected" "$(echo "$BOTH_OUT" | tail -5)"
fi
rm -f "$TEST_DIR/resources/both.yaml"

"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$API_PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow judges - server failed to start" "$LOG_FILE"
    return 0 2>/dev/null || exit 0
fi
API_PORT="$KDEPS_BOUND_PORT"

RESP=$(curl -sf --max-time 20 "http://127.0.0.1:${API_PORT}/notes" 2>&1)
NOTES=$(echo "$RESP" | python3 -c 'import json,sys; d=json.load(sys.stdin); d=d.get("data",d); print(json.dumps(d.get("notes")))' 2>/dev/null)
PRESET=$(echo "$RESP" | python3 -c 'import json,sys; d=json.load(sys.stdin); d=d.get("data",d); print(json.dumps(d.get("preset")))' 2>/dev/null)
GRADED=$(echo "$RESP" | python3 -c 'import json,sys; d=json.load(sys.stdin); d=d.get("data",d); print(json.dumps(d.get("graded")))' 2>/dev/null)

if echo "$NOTES" | grep -q "REVISED notes" && ! echo "$NOTES" | grep -q "DRAFT"; then
    test_passed "Workflow judges - rejected draft is revised with the judges' feedback"
else
    test_failed "Workflow judges - rejected draft is revised with the judges' feedback" "resp=$RESP"
fi

if echo "$GRADED" | grep -q "REVISED notes"; then
    test_passed "Workflow judges - a grader-only judge rejects without a model call"
else
    test_failed "Workflow judges - a grader-only judge rejects without a model call" "resp=$RESP"
fi

if echo "$PRESET" | grep -q "REVISED notes" && grep -q "scope, clarity, failure-modes, readiness" "$LOG_FILE"; then
    test_passed "Workflow judges - judgePreset runs the preset's judges"
else
    test_failed "Workflow judges - judgePreset runs the preset's judges" "resp=$RESP"
fi
