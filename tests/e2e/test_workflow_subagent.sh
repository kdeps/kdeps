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

# E2E: a workflow chat lists subagent by name. The model hands it a task; a
# fresh chat call with only the chat's read-only tools (calculator) runs it
# and its report comes back as the tool result. A write tool is refused.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing subagent in workflow chat..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow subagent - python3 not available"
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
  name: subagent-e2e
  version: "1.0.0"
  targetActionId: response
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${API_PORT}
    routes:
      - path: /review
        methods: [GET]
      - path: /refused
        methods: [GET]
YAML

cat > "$TEST_DIR/resources/review.yaml" <<'YAML'
actionId: review
name: Subagent chat
validations:
  routes: [/review]
chat:
  model:
    name: fake
  context:
    prompt: 'CALLJSON subagent {"task": "CALLJSON calculator {\"expression\": \"6 * 7\"}"}'
  tools:
    list:
      - name: subagent
      - name: calculator
YAML

cat > "$TEST_DIR/resources/refused.yaml" <<'YAML'
actionId: refused
name: Subagent asked for a write tool
validations:
  routes: [/refused]
chat:
  model:
    name: fake
  context:
    prompt: 'CALLJSON subagent {"task": "write it", "tools": "write_file"}'
  tools:
    list:
      - name: subagent
      - name: write_file
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [review, refused]
apiResponse:
  success: true
  response:
    review: "{{ output('review') }}"
    refused: "{{ output('refused') }}"
YAML

if "$KDEPS_BIN" validate "$TEST_DIR/workflow.yaml" &>/dev/null; then
    test_passed "Workflow subagent - a chat listing subagent validates"
else
    test_failed "Workflow subagent - a chat listing subagent validates" "$("$KDEPS_BIN" validate "$TEST_DIR/workflow.yaml" 2>&1 | tail -5)"
fi

"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$API_PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow subagent - server failed to start" "$LOG_FILE"
    return 0 2>/dev/null || exit 0
fi
API_PORT="$KDEPS_BOUND_PORT"

RESP=$(curl -s --max-time 30 "http://127.0.0.1:${API_PORT}/review" 2>&1)
if echo "$RESP" | grep -q "TOOL RESULT TOOL RESULT 42"; then
    test_passed "Workflow subagent - the subagent calls calculator and reports back"
else
    test_failed "Workflow subagent - the subagent calls calculator and reports back" "resp=$RESP"
fi

RESP=$(curl -s --max-time 30 "http://127.0.0.1:${API_PORT}/refused" 2>&1)
if echo "$RESP" | grep -q "not a read-only tool here: write_file"; then
    test_passed "Workflow subagent - a write tool is refused"
else
    test_failed "Workflow subagent - a write tool is refused" "resp=$RESP"
fi
