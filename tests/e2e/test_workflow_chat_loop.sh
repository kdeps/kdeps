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

# E2E: a workflow chat with a loop: block runs the agent loop. The model calls
# the read_file built-in tool, the reply reaches the next resource, and a
# second request with the same session id continues that conversation.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat loop..."

if ! command -v python3 &> /dev/null || [ -z "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "Workflow chat loop - needs python3 and the mock LLM"
    return 0 2>/dev/null || exit 0
fi

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT
echo "LOOPSECRET 7" > "$TEST_DIR/note.txt"

PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
cat > "$TEST_DIR/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: chat-loop-e2e
  version: "1.0.0"
  targetActionId: response
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${PORT}
    routes:
      - path: /ask
        methods: [POST]
YAML

cat > "$TEST_DIR/resources/ask.yaml" <<'YAML'
actionId: ask
name: Ask
chat:
  model:
    name: llama3.2:1b
  context:
    prompt: "{{ get('q') }}"
  loop:
    tools: [read_file]
    maxRounds: 4
    session: "{{ get('s') }}"
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [ask]
apiResponse:
  success: true
  response:
    answer: "{{ output('ask') }}"
YAML

if ! "$KDEPS_BIN" validate "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1; then
    test_failed "Workflow chat loop - chat.loop validates" "$(tail -5 "$LOG_FILE")"
    return 1 2>/dev/null || exit 1
fi
test_passed "Workflow chat loop - chat.loop validates"

# read_file is fenced to the workspace: the test folder.
KDEPS_WORKSPACE_ROOT="$TEST_DIR" "$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow chat loop: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi

BODY=$(python3 -c 'import json,sys; print(json.dumps({"q": "LOOPE2E-FIRST CALLJSON read_file " + json.dumps({"file_path": sys.argv[1]}), "s": "one"}))' "$TEST_DIR/note.txt")
RESP=$(curl -s --max-time 60 -H "Content-Type: application/json" -d "$BODY" "http://127.0.0.1:${KDEPS_BOUND_PORT}/ask" 2>&1)
if echo "$RESP" | grep -qF "TOOL RESULT" && echo "$RESP" | grep -qF "LOOPSECRET 7"; then
    test_passed "Workflow chat loop - the loop calls read_file and the reply reaches the next resource"
else
    test_failed "Workflow chat loop - the loop calls read_file and the reply reaches the next resource" "resp=$RESP"
fi

curl -s --max-time 60 -H "Content-Type: application/json" -d '{"q": "LOOPE2E-SECOND", "s": "one"}' \
    "http://127.0.0.1:${KDEPS_BOUND_PORT}/ask" > /dev/null 2>&1
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null

# The second request carries the first one's conversation.
if python3 - "$E2E_MOCK_LLM_LOG" <<'PY'
import json, sys
for line in open(sys.argv[1], encoding="utf-8"):
    try:
        msgs = json.loads(line).get("messages", [])
    except ValueError:
        continue
    texts = [m.get("content") if isinstance(m.get("content"), str) else json.dumps(m.get("content")) for m in msgs]
    if texts and "LOOPE2E-SECOND" in texts[-1] and any("LOOPE2E-FIRST" in t for t in texts[:-1]):
        sys.exit(0)
sys.exit(1)
PY
then
    test_passed "Workflow chat loop - the same session id continues the conversation"
else
    test_failed "Workflow chat loop - the same session id continues the conversation" "no request carried the first turn"
fi
