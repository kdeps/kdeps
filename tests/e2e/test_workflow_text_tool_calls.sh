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

# E2E: a workflow chat runs a tool call the model wrote as text: an <invoke>
# block (CALL) or DeepSeek DSML markup (CALLDSML). The calculator result only
# reaches the mock in a tool message when the call really ran.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow text-written tool calls..."

if [ -z "${E2E_MOCK_LLM_LOG:-}" ] || [ ! -f "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "Workflow text tool calls - the shared mock LLM is not running"
    return 0 2>/dev/null || exit 0
fi

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT

cat > "$TEST_DIR/resources/invoke.yaml" <<'YAML'
actionId: invoke
name: Invoke chat
chat:
  model:
    name: fake
  context:
    prompt: 'CALL calculator {"expression": "6*7*1000"}'
  tools:
    list:
      - name: calculator
YAML

cat > "$TEST_DIR/resources/dsml.yaml" <<'YAML'
actionId: dsml
name: DSML chat
chat:
  model:
    name: fake
  context:
    prompt: 'CALLDSML calculator {"expression": "6*7*100"}'
  tools:
    list:
      - name: calculator
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [invoke, dsml]
apiResponse:
  success: true
  response:
    invoke: "{{ output('invoke') }}"
    dsml: "{{ output('dsml') }}"
YAML

PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
cat > "$TEST_DIR/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: text-tool-calls-e2e
  version: "1.0.0"
  targetActionId: response
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${PORT}
    routes:
      - path: /run
        methods: [GET]
YAML

before=$(wc -c < "$E2E_MOCK_LLM_LOG")
"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow text tool calls: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi
RESP=$(curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}/run" 2>&1)
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null
SENT=$(tail -c +"$((before + 1))" "$E2E_MOCK_LLM_LOG")

if grep -q "42000" <<<"$SENT"; then
    test_passed "Workflow text tool calls - an <invoke> block runs the tool"
else
    test_failed "Workflow text tool calls - an <invoke> block runs the tool" "resp=$RESP"
fi

if grep -q "4200" <<<"$(grep -v 42000 <<<"$SENT")"; then
    test_passed "Workflow text tool calls - DSML markup runs the tool"
else
    test_failed "Workflow text tool calls - DSML markup runs the tool" "resp=$RESP"
fi

if grep -q "TOOL DONE" <<<"$RESP" && ! grep -q "｜" <<<"$RESP"; then
    test_passed "Workflow text tool calls - the answer follows the tool result"
else
    test_failed "Workflow text tool calls - the answer follows the tool result" "resp=$RESP"
fi
