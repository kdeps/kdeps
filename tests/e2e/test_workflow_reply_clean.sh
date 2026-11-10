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

# E2E: a workflow chat that used tools returns a clean answer. Markup the
# model leaked after the tool result ([MEMORY:] markers, a self-written
# <tool_response>, <File> wrappers) is stripped, as in agent mode.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat reply cleanup..."

if [ -z "${E2E_MOCK_LLM_LOG:-}" ] || [ ! -f "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "Workflow reply cleanup - the shared mock LLM is not running"
    return 0 2>/dev/null || exit 0
fi

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT

cat > "$TEST_DIR/resources/ask.yaml" <<'YAML'
actionId: ask
name: Leaky chat
chat:
  model:
    name: fake
  context:
    prompt: 'LEAKJSON calculator {"expression": "6*7"}'
  tools:
    list:
      - name: calculator
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [ask]
apiResponse:
  success: true
  response:
    answer: "{{ output('ask').message.content }}"
YAML

PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
cat > "$TEST_DIR/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: reply-clean-e2e
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

"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow reply cleanup: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi
RESP=$(curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}/run" 2>&1)
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null

if grep -q "LEAKED 42" <<<"$RESP" && grep -q "See a.txt" <<<"$RESP"; then
    test_passed "Workflow reply cleanup - the answer keeps its real content"
else
    test_failed "Workflow reply cleanup - the answer keeps its real content" "resp=$RESP"
fi

if grep -qE "MEMORY:|tool_response|<File>|fake" <<<"$RESP"; then
    test_failed "Workflow reply cleanup - leaked markup is stripped" "resp=$RESP"
else
    test_passed "Workflow reply cleanup - leaked markup is stripped"
fi
