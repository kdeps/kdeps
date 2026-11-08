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

# E2E: a workflow chat lists a kdeps tool by name (calculator) with no
# script or mcp. The tool's description, parameters and code come from its
# asset; the mock LLM calls it and answers with the result.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing kdeps tools in workflow chat..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow kdeps tools - python3 not available"
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
  name: kdeps-tools-e2e
  version: "1.0.0"
  targetActionId: response
settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${API_PORT}
    routes:
      - path: /calc
        methods: [GET]
      - path: /missing
        methods: [GET]
YAML

cat > "$TEST_DIR/resources/calc.yaml" <<'YAML'
actionId: calc
name: Calculator chat
validations:
  routes: [/calc]
chat:
  model: fake
  prompt: 'CALLJSON calculator {"expression": "6 * 7"}'
  tools:
    - name: calculator
YAML

cat > "$TEST_DIR/resources/missing.yaml" <<'YAML'
actionId: missing
name: Unknown tool chat
validations:
  routes: [/missing]
chat:
  model: fake
  prompt: "hello"
  tools:
    - name: no_such_tool
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
requires: [calc, missing]
apiResponse:
  success: true
  response:
    calc: "{{ output('calc') }}"
YAML

if "$KDEPS_BIN" validate "$TEST_DIR/workflow.yaml" &>/dev/null; then
    test_passed "Workflow kdeps tools - a chat listing a kdeps tool by name validates"
else
    test_failed "Workflow kdeps tools - a chat listing a kdeps tool by name validates" "$("$KDEPS_BIN" validate "$TEST_DIR/workflow.yaml" 2>&1 | tail -5)"
fi

"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$API_PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow kdeps tools - server failed to start" "$LOG_FILE"
    return 0 2>/dev/null || exit 0
fi
API_PORT="$KDEPS_BOUND_PORT"

RESP=$(curl -s --max-time 20 "http://127.0.0.1:${API_PORT}/calc" 2>&1)
if echo "$RESP" | grep -q "TOOL RESULT 42"; then
    test_passed "Workflow kdeps tools - the model calls calculator and answers with its result"
else
    test_failed "Workflow kdeps tools - the model calls calculator and answers with its result" "resp=$RESP"
fi

RESP=$(curl -s --max-time 20 "http://127.0.0.1:${API_PORT}/missing" 2>&1)
if echo "$RESP" | grep -q '"success":false' && grep -q "no_such_tool.*no script, mcp or kdeps tool by that name" "$LOG_FILE"; then
    test_passed "Workflow kdeps tools - an unknown tool name fails the resource"
else
    test_failed "Workflow kdeps tools - an unknown tool name fails the resource" "resp=$RESP"
fi
