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

# E2E: a workflow chat calls agent mode's memory and knowledge tools by name
# (memory_save, knowledge_recall), with no script or mcp of its own.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat memory and knowledge tools..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow chat memory tools - python3 not available"
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
  name: chat-memory-e2e
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

cat > "$TEST_DIR/resources/save.yaml" <<'YAML'
actionId: save
name: Save
guardrails:
  routes: [/ask]
chat:
  model:
    name: fake
  context:
    prompt: 'CALLJSON memory_save {"key": "e2e-color", "value": "teal"}'
  tools:
    list:
      - name: memory_save
YAML

cat > "$TEST_DIR/resources/recall.yaml" <<'YAML'
actionId: recall
name: Recall
guardrails:
  routes: [/ask]
workflow:
  requires: [save]
chat:
  model:
    name: fake
  context:
    prompt: 'CALLJSON knowledge_recall {"query": "billing decision"}'
  tools:
    list:
      - name: knowledge_recall
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
workflow:
  requires: [save, recall]
apiResponse:
  success: true
  response:
    saved: "{{ output('save') }}"
    recalled: "{{ output('recall') }}"
YAML

"$KDEPS_BIN" run --memory "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow chat memory tools: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi
RESP=$(curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}/ask" 2>&1)
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null

if echo "$RESP" | grep -q 'TOOL RESULT Saved' || echo "$RESP" | grep -q 'TOOL RESULT .*e2e-color'; then
    test_passed "Workflow chat memory tools - memory_save writes the run's memory store (kdeps run --memory)"
else
    test_failed "Workflow chat memory tools - memory_save writes the run's memory store" "resp=$RESP"
fi
if echo "$RESP" | grep -q 'TOOL RESULT No notes in the knowledge vault'; then
    test_passed "Workflow chat memory tools - knowledge_recall answers from the vault"
else
    test_failed "Workflow chat memory tools - knowledge_recall answers from the vault" "resp=$RESP"
fi
