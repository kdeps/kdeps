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

# E2E: chat.refine rewrites a workflow chat's prompt with the refine-system
# harness before the real call, which then answers the rewrite. A chat

# E2E: a workflow chat's kdeps tool calls pass the permission policy, as the
# agent loop's do: in a read-only run (KDEPS_PERMISSION_MODE=read-only) a
# write_file call is refused and the file is not written.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat tool permission..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow chat tool permission - python3 not available"
    return 0 2>/dev/null || exit 0
fi

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT
TARGET="$TEST_DIR/blocked.txt"

PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
cat > "$TEST_DIR/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: chat-permission-e2e
  version: "1.0.0"
  targetActionId: response
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${PORT}
    routes:
      - path: /ask
        methods: [GET]
YAML

cat > "$TEST_DIR/resources/ask.yaml" <<YAML
actionId: ask
name: Ask
guardrails:
  routes: [/ask]
chat:
  model:
    name: fake
  context:
    prompt: 'CALLJSON write_file {"file_path": "${TARGET}", "content": "x"}'
  tools:
    list:
      - name: write_file
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
workflow:
  requires: [ask]
apiResponse:
  success: true
  response:
    answer: "{{ output('ask') }}"
YAML

KDEPS_PERMISSION_MODE=read-only "$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow chat tool permission: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi
RESP=$(curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}/ask" 2>&1)
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null

if echo "$RESP" | grep -qF 'requires permission' && [ ! -e "$TARGET" ]; then
    test_passed "Workflow chat tool permission - a read-only run refuses write_file"
else
    test_failed "Workflow chat tool permission - a read-only run refuses write_file" "resp=$RESP exists=$([ -e "$TARGET" ] && echo yes || echo no)"
fi
