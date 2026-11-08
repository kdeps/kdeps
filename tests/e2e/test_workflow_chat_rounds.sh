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

# E2E: a workflow chat's tool-call guard comes from events. Default rounds
# are chat-max-rounds (5); a ~/.kdeps/events override changes them. A budget
# event (tools: + distinctCalls) blocks new distinct calls, and a call the
# model keeps repeating stops at identical-tool-calls (3), run only once. A
# tool-result-truncate override cuts what the model sees of a result.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing the workflow chat tool guard..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow chat rounds - python3 not available"
    return 0 2>/dev/null || exit 0
fi

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources" "$TEST_DIR/home/.kdeps/events" "$TEST_DIR/budget/.kdeps/events" "$TEST_DIR/trunc/.kdeps/events"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT

cat > "$TEST_DIR/resources/loop.yaml" <<'YAML'
actionId: loop
name: Looping chat
validations:
  routes: [/loop]
chat:
  model: fake
  prompt: "LOOPJSON calculator"
  tools:
    - name: calculator
YAML

cat > "$TEST_DIR/resources/repeat.yaml" <<'YAML'
actionId: repeat
name: Repeating chat
validations:
  routes: [/repeat]
chat:
  model: fake
  prompt: 'REPEATJSON calculator {"expression": "1 + 1"}'
  maxToolRounds: 10
  tools:
    - name: calculator
YAML

cat > "$TEST_DIR/resources/big.yaml" <<'YAML'
actionId: big
name: Big result chat
validations:
  routes: [/big]
chat:
  model: fake
  prompt: 'CALLJSON calculator {"expression": "10**40"}'
  tools:
    - name: calculator
YAML

cat > "$TEST_DIR/trunc/.kdeps/events/tool-result-truncate.yaml" <<'YAML'
name: tool-result-truncate
on:
  bytes: 12
run: truncate
YAML

cat > "$TEST_DIR/budget/.kdeps/events/calc-budget.yaml" <<'YAML'
name: calc-budget
tools: [calculator]
on:
  distinctCalls: 2
run: block
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
requires: [loop, repeat, big]
apiResponse:
  success: true
  response:
    loop: "{{ output('loop') }}"
    repeat: "{{ output('repeat') }}"
    big: "{{ output('big') }}"
YAML

cat > "$TEST_DIR/home/.kdeps/events/chat-max-rounds.yaml" <<'YAML'
name: chat-max-rounds
on:
  rounds: 2
run: force_answer
YAML

# run_rounds <label> [home] [path]: serve the workflow, call path once (/loop), print the reply.
run_rounds() {
    local port
    port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
    cat > "$TEST_DIR/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: chat-rounds-e2e
  version: "1.0.0"
  targetActionId: response
settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${port}
    routes:
      - path: /loop
        methods: [GET]
      - path: /repeat
        methods: [GET]
      - path: /big
        methods: [GET]
YAML
    if [ -n "${2:-}" ]; then
        HOME="$2" USERPROFILE="$(to_native_path "$2")" "$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
    else
        "$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
    fi
    KDEPS_PID=$!
    if ! wait_for_kdeps_port "$port" 20 "" "$LOG_FILE"; then
        fail_server_startup "Workflow chat rounds - $1: server failed to start" "$LOG_FILE"
        return 1
    fi
    curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}${3:-/loop}" 2>&1
    kill "$KDEPS_PID" 2>/dev/null
    wait "$KDEPS_PID" 2>/dev/null
}

RESP=$(run_rounds "default")
if echo "$RESP" | grep -q "LOOP DONE 5"; then
    test_passed "Workflow chat rounds - the chat-max-rounds event caps a chat at 5 tool rounds"
else
    test_failed "Workflow chat rounds - the chat-max-rounds event caps a chat at 5 tool rounds" "resp=$RESP"
fi

RESP=$(run_rounds "override" "$TEST_DIR/home")
if echo "$RESP" | grep -q "LOOP DONE 2"; then
    test_passed "Workflow chat rounds - a ~/.kdeps/events override changes the cap"
else
    test_failed "Workflow chat rounds - a ~/.kdeps/events override changes the cap" "resp=$RESP"
fi

RESP=$(run_rounds "budget" "$TEST_DIR/budget")
if echo "$RESP" | grep -q "LOOP DONE 5 BLOCKED 3"; then
    test_passed "Workflow chat rounds - a budget event blocks calls past its distinctCalls"
else
    test_failed "Workflow chat rounds - a budget event blocks calls past its distinctCalls" "resp=$RESP"
fi

RESP=$(run_rounds "repeat" "" "/repeat")
if echo "$RESP" | grep -q "REPEAT DONE 3"; then
    test_passed "Workflow chat rounds - a repeated call stops at identical-tool-calls"
else
    test_failed "Workflow chat rounds - a repeated call stops at identical-tool-calls" "resp=$RESP"
fi

RESP=$(run_rounds "truncate" "$TEST_DIR/trunc" "/big")
if echo "$RESP" | grep -q "tool result truncated: 41 bytes total"; then
    test_passed "Workflow chat rounds - a tool-result-truncate override cuts a tool result"
else
    test_failed "Workflow chat rounds - a tool-result-truncate override cuts a tool result" "resp=$RESP"
fi
