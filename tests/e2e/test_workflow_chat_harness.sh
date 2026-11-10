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

# E2E: chat.harness adds agent mode's harness sections to a workflow chat's
# system prompt; a chat without it gets none; an unknown name fails the chat.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat harness sections..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow chat harness - python3 not available"
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
  name: chat-harness-e2e
  version: "1.0.0"
  targetActionId: response
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${PORT}
    routes:
      - path: /ask
        methods: [GET]
      - path: /bad
        methods: [GET]
YAML

cat > "$TEST_DIR/resources/with.yaml" <<'YAML'
actionId: with
name: With sections
validations:
  routes: [/ask]
chat:
  model:
    name: fake
  context:
    prompt: "HARNESSCHECK"
  harness:
    sections: [accuracy, honesty]
YAML

cat > "$TEST_DIR/resources/without.yaml" <<'YAML'
actionId: without
name: Without sections
validations:
  routes: [/ask]
chat:
  model:
    name: fake
  context:
    prompt: "HARNESSCHECK"
YAML

cat > "$TEST_DIR/resources/bad.yaml" <<'YAML'
actionId: bad
name: Unknown section
validations:
  routes: [/bad]
chat:
  model:
    name: fake
  context:
    prompt: "HARNESSCHECK"
  harness:
    sections: [no-such-section]
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [with, without, bad]
apiResponse:
  success: true
  response:
    with: "{{ output('with') }}"
    without: "{{ output('without') }}"
YAML

"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow chat harness: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi
RESP=$(curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}/ask" 2>&1)
BAD=$(curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}/bad" 2>&1)
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null

if echo "$RESP" | grep -q "HARNESS accuracy,honesty" && echo "$RESP" | grep -q "HARNESS none"; then
    test_passed "Workflow chat harness - chat.harness adds the named sections, and only to that chat"
else
    test_failed "Workflow chat harness - chat.harness adds the named sections, and only to that chat" "resp=$RESP"
fi

if echo "$BAD" | grep -q '"success":false' && grep -q 'no preamble section named \\\?"no-such-section' "$LOG_FILE"; then
    test_passed "Workflow chat harness - an unknown section fails the chat"
else
    test_failed "Workflow chat harness - an unknown section fails the chat" "resp=$BAD"
fi
