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

# E2E: a workflow chat checks its request against the provider-body-limit-*
# event before calling the model. Under the default limit the model answers;
# a ~/.kdeps/events override below the request size fails the chat with an
# actionable error (in the server log) and the model is never called.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing the workflow chat request body limit..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow body limit - python3 not available"
    return 0 2>/dev/null || exit 0
fi

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources" "$TEST_DIR/small/.kdeps/events"
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT

cat > "$TEST_DIR/resources/ask.yaml" <<'YAML'
actionId: ask
name: Ask
validations:
  routes: [/ask]
chat:
  model: fake
  prompt: "BODYLIMIT hello"
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
requires: [ask]
apiResponse:
  success: true
  response:
    answer: "{{ output('ask') }}"
YAML

cat > "$TEST_DIR/small/.kdeps/events/provider-body-limit-openai.yaml" <<'YAML'
name: provider-body-limit-openai
on:
  bytes: 20
run: reject
YAML

# run_ask <label> [home]: serve the workflow, call /ask once, print the reply.
run_ask() {
    local port
    port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
    cat > "$TEST_DIR/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: body-limit-e2e
  version: "1.0.0"
  targetActionId: response
settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${port}
    routes:
      - path: /ask
        methods: [GET]
YAML
    if [ -n "${2:-}" ]; then
        HOME="$2" USERPROFILE="$(to_native_path "$2")" "$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
    else
        "$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
    fi
    KDEPS_PID=$!
    if ! wait_for_kdeps_port "$port" 20 "" "$LOG_FILE"; then
        fail_server_startup "Workflow body limit - $1: server failed to start" "$LOG_FILE"
        return 1
    fi
    curl -s --max-time 30 "http://127.0.0.1:${KDEPS_BOUND_PORT}/ask" 2>&1
    kill "$KDEPS_PID" 2>/dev/null
    wait "$KDEPS_PID" 2>/dev/null
}

mock_calls() { grep -c "BODYLIMIT" "${E2E_MOCK_LLM_LOG:-/dev/null}" 2>/dev/null || echo 0; }

BEFORE=$(mock_calls)
RESP=$(run_ask "default")
if echo "$RESP" | grep -q "MOCK REPLY"; then
    test_passed "Workflow body limit - a request under the limit reaches the model"
else
    test_failed "Workflow body limit - a request under the limit reaches the model" "resp=$RESP"
fi

MID=$(mock_calls)
RESP=$(run_ask "small" "$TEST_DIR/small")
AFTER=$(mock_calls)
if echo "$RESP" | grep -q "\"success\":false" && grep -q "exceeds openai limit of 20 bytes" "$LOG_FILE" && [ "$AFTER" = "$MID" ]; then
    test_passed "Workflow body limit - a request over the limit fails before the model call"
else
    test_failed "Workflow body limit - a request over the limit fails before the model call" "resp=$RESP before=$BEFORE mid=$MID after=$AFTER"
fi
