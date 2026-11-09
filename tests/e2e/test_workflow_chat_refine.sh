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
# without refine sends its prompt as written.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat prompt refine..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow chat refine - python3 not available"
    return 0 2>/dev/null || exit 0
fi

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT

PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
cat > "$TEST_DIR/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: chat-refine-e2e
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

cat > "$TEST_DIR/resources/refined.yaml" <<'YAML'
actionId: refined
name: Refined
guardrails:
  routes: [/ask]
chat:
  model:
    name: fake
  context:
    prompt: "REFINEME please write the quarterly status report for the platform team"
  harness:
    refine: true
YAML

cat > "$TEST_DIR/resources/plain.yaml" <<'YAML'
actionId: plain
name: Plain
guardrails:
  routes: [/ask]
chat:
  model:
    name: fake
  context:
    prompt: "REFINEME please write the quarterly status report for the platform team"
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
workflow:
  requires: [refined, plain]
apiResponse:
  success: true
  response:
    refined: "{{ output('refined') }}"
    plain: "{{ output('plain') }}"
YAML

"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow chat refine: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi
RESP=$(curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}/ask" 2>&1)
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null

if echo "$RESP" | grep -q "GOT REFINED" && echo "$RESP" | grep -q "GOT ORIGINAL"; then
    test_passed "Workflow chat refine - refine: true answers the rewritten prompt, and only that chat"
else
    test_failed "Workflow chat refine - refine: true answers the rewritten prompt, and only that chat" "resp=$RESP"
fi
