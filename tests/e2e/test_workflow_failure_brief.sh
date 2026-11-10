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

# E2E: a failed workflow tool call gets a failure brief, as in agent mode:
# the folder of the missing path and the instruction to fix the cause and
# call again, in the tool message the model reads.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow failure briefs..."

if [ -z "${E2E_MOCK_LLM_LOG:-}" ] || [ ! -f "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "Workflow failure brief - the shared mock LLM is not running"
    return 0 2>/dev/null || exit 0
fi

TEST_DIR=$(cd "$(mktemp -d)" && pwd -P)
mkdir -p "$TEST_DIR/resources" "$TEST_DIR/data"
echo hi > "$TEST_DIR/data/present.txt"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT
MISSING="$(to_native_path "$TEST_DIR/data/missing.txt" | sed 's|\\|/|g')"

cat > "$TEST_DIR/resources/ask.yaml" <<YAML
actionId: ask
name: Failing read
chat:
  model:
    name: fake
  context:
    prompt: 'CALLJSON read_file {"path": "${MISSING}"}'
  tools:
    list:
      - name: read_file
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
  name: failure-brief-e2e
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
    fail_server_startup "Workflow failure brief: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi
RESP=$(curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}/run" 2>&1)
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null

if grep -q "missing.txt does not exist" <<<"$RESP" && grep -q "present.txt" <<<"$RESP"; then
    test_passed "Workflow failure brief - a missing path lists its folder"
else
    test_failed "Workflow failure brief - a missing path lists its folder" "resp=$RESP"
fi

if grep -q "call again now" <<<"$RESP"; then
    test_passed "Workflow failure brief - the model is told to fix the cause and call again"
else
    test_failed "Workflow failure brief - the model is told to fix the cause and call again" "resp=$RESP"
fi
