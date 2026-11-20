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
# E2E: harness.efficiency soft-stops a workflow chat that keeps reading: the
# model gets the hidden note and no read tools. harness.promptMode: summary
# sends the harness section summaries after the first request. A chat without
# harness.efficiency reads until the round cap.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat efficiency and prompt mode..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow chat efficiency - python3 not available"
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
  name: chat-efficiency-e2e
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

write_chat() { # actionId harness-extra
    cat > "$TEST_DIR/resources/$1.yaml" <<YAML
actionId: $1
name: $1
validations:
  routes: [/ask]
chat:
  model:
    name: fake
  context:
    prompt: "EFFJSON look around"
  harness:
    sections: [accuracy]
$2
  tools:
    maxRounds: 12
    list:
      - name: list_files
      - name: write_file
YAML
}
write_chat governed "    efficiency: true
    promptMode: summary"
write_chat ungoverned ""

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [governed, ungoverned]
apiResponse:
  success: true
  response:
    governed: "{{ output('governed') }}"
    ungoverned: "{{ output('ungoverned') }}"
YAML

(cd "$TEST_DIR" && "$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1) &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow chat efficiency: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi
RESP=$(curl -s --max-time 60 "http://127.0.0.1:${KDEPS_BOUND_PORT}/ask" 2>&1)
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null

field() { # name: the answer of one chat, from the JSON response
    echo "$RESP" | python3 -c 'import json,sys
d=json.load(sys.stdin)
v=d.get("data",d)
v=v.get("response",v) if isinstance(v,dict) else v
print(json.dumps(v.get(sys.argv[1]) if isinstance(v,dict) else v))' "$1" 2>/dev/null
}
GOVERNED=$(field governed)
UNGOVERNED=$(field ungoverned)

if echo "$GOVERNED" | grep -q "EFF STOPPED"; then
    test_passed "Workflow chat efficiency - harness.efficiency soft-stops a run of reads"
else
    test_failed "Workflow chat efficiency - harness.efficiency soft-stops a run of reads" "resp=$RESP"
fi

if echo "$GOVERNED" | grep -q "EFF STOPPED SUMMARY"; then
    test_passed "Workflow chat efficiency - harness.promptMode: summary sends the section summaries"
else
    test_failed "Workflow chat efficiency - harness.promptMode: summary sends the section summaries" "resp=$RESP"
fi

if echo "$UNGOVERNED" | grep -q "EFF NOT STOPPED"; then
    test_passed "Workflow chat efficiency - a chat without harness.efficiency is not stopped"
else
    test_failed "Workflow chat efficiency - a chat without harness.efficiency is not stopped" "resp=$RESP"
fi
