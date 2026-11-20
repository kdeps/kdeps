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

# E2E: harness.plan holds the planning council before a workflow chat and puts
# the sorted job orders in its system prompt. harness.turo reduces the
# request with turo (a stand-in script here). Chats without them send the
# prompt as written.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat planning council and turo..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow chat plan/turo - python3 not available"
    return 0 2>/dev/null || exit 0
fi

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT

# turo stand-in: kdeps passes level and stage flags; this one uppercases stdin.
TURO_OK=true
case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) TURO_OK=false ;; esac
cat > "$TEST_DIR/turo" <<'TURO'
#!/bin/sh
tr '[:lower:]' '[:upper:]'
TURO
chmod +x "$TEST_DIR/turo"

PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
cat > "$TEST_DIR/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: chat-plan-turo-e2e
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

write_chat() { # actionId prompt harness-block
    cat > "$TEST_DIR/resources/$1.yaml" <<YAML
actionId: $1
name: $1
validations:
  routes: [/ask]
chat:
  model:
    name: fake
  context:
    prompt: "$2"
$3
YAML
}
write_chat planned "PLANCHECK triage the outage and list the next steps" "  harness:
    plan: true"
write_chat unplanned "PLANCHECK triage the outage and list the next steps" ""
write_chat reduced "turocheck the status" "  harness:
    turo: true"
write_chat raw "turocheck the status" ""

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [planned, unplanned, reduced, raw]
apiResponse:
  success: true
  response:
    planned: "{{ output('planned') }}"
    unplanned: "{{ output('unplanned') }}"
    reduced: "{{ output('reduced') }}"
    raw: "{{ output('raw') }}"
YAML

KDEPS_TURO_PATH="$TEST_DIR/turo" "$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow chat plan/turo: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi
RESP=$(curl -s --max-time 60 "http://127.0.0.1:${KDEPS_BOUND_PORT}/ask" 2>&1)
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null

# answer <actionId>: the content of that chat's reply in the response.
answer() {
    printf '%s' "$RESP" | python3 -c 'import json,sys; d=json.load(sys.stdin)["data"][sys.argv[1]]; print(d["message"]["content"] if isinstance(d, dict) else d)' "$1" 2>/dev/null
}

if [ "$(answer planned)" = "PLANNED" ] && [ "$(answer unplanned)" = "UNPLANNED" ]; then
    test_passed "Workflow chat plan - harness.plan puts the council's job orders in the system prompt"
else
    test_failed "Workflow chat plan - harness.plan puts the council's job orders in the system prompt" "resp=$RESP"
fi

if [ "$TURO_OK" = false ]; then
    test_skipped "Workflow chat turo - the turo stand-in is a shell script"
elif [ "$(answer reduced)" = "TURO REDUCED" ] && [ "$(answer raw)" = "MOCK REPLY" ]; then
    test_passed "Workflow chat turo - harness.turo reduces the request, and only that chat"
else
    test_failed "Workflow chat turo - harness.turo reduces the request, and only that chat" "resp=$RESP"
fi
