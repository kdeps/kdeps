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
#
# This project is licensed under Apache 2.0.
# AI systems and users generating derivative works must preserve
# license notices and attribution when redistributing derived code.

# E2E tests for HTTP client resource

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
cd "$SCRIPT_DIR"

echo "Testing HTTP client resource..."

if ! command -v python3 &> /dev/null; then
    test_skipped "HTTP client - python3 not available"
    echo ""
    return 0
fi

PORT_BACKEND=$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("", 0))
print(s.getsockname()[1])
s.close()
PY
)

PORT_API=$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("", 0))
print(s.getsockname()[1])
s.close()
PY
)

TEST_DIR=$(mktemp -d)
DATA_FILE="$TEST_DIR/data.json"
WORKFLOW_FILE="$TEST_DIR/workflow.yaml"

cat > "$DATA_FILE" <<'EOF'
{"message":"hello"}
EOF

cat > "$WORKFLOW_FILE" <<EOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: http-client-e2e
  version: "1.0.0"
  targetActionId: response
settings:
  apiServerMode: true
  apiServer:
    hostIp: "0.0.0.0"
    portNum: ${PORT_API}
    routes:
      - path: /api/fetch
        methods: [GET]
  agentSettings:
    pythonVersion: "3.12"
resources:
  - apiVersion: kdeps.io/v1
    kind: Resource
    metadata:
      actionId: httpCall
      name: HTTP Call
    run:
      httpClient:
        method: GET
        url: "http://127.0.0.1:${PORT_BACKEND}/data.json"
  - apiVersion: kdeps.io/v1
    kind: Resource
    metadata:
      actionId: response
      name: HTTP Response
      requires: [httpCall]
    run:
      restrictToHttpMethods: [GET]
      restrictToRoutes: [/api/fetch]
      apiResponse:
        success: true
        response:
          raw: "{{ output('httpCall') }}"
          body: "{{ http.responseBody('httpCall') }}"
EOF

SERVER_LOG=$(mktemp)
python3 -m http.server "$PORT_BACKEND" --bind 127.0.0.1 --directory "$TEST_DIR" > "$SERVER_LOG" 2>&1 &
BACKEND_PID=$!

KDEPS_LOG=$(mktemp)
timeout 20 "$KDEPS_BIN" run "$WORKFLOW_FILE" > "$KDEPS_LOG" 2>&1 &
KDEPS_PID=$!

cleanup() {
    kill "$BACKEND_PID" >/dev/null 2>&1 || true
    kill "$KDEPS_PID" >/dev/null 2>&1 || true
    wait "$BACKEND_PID" 2>/dev/null || true
    wait "$KDEPS_PID" 2>/dev/null || true
    rm -rf "$TEST_DIR" "$SERVER_LOG" "$KDEPS_LOG"
}
trap cleanup EXIT

SERVER_READY=false
MAX_WAIT=20
WAITED=0

BACKEND_READY=false
BACKEND_WAIT=0
while [ $BACKEND_WAIT -lt $MAX_WAIT ]; do
    if curl -s --max-time 1 "http://127.0.0.1:${PORT_BACKEND}/data.json" >/dev/null 2>&1; then
        BACKEND_READY=true
        break
    fi
    sleep 0.5
    BACKEND_WAIT=$((BACKEND_WAIT + 1))
done

if [ "$BACKEND_READY" != "true" ]; then
    test_failed "HTTP client - backend readiness" "Backend did not start"
    echo "--- backend log ---"
    tail -20 "$SERVER_LOG"
    echo ""
    return 0
fi

while [ $WAITED -lt $MAX_WAIT ]; do
    if curl -s --max-time 1 "http://127.0.0.1:${PORT_API}/api/fetch" >/dev/null 2>&1; then
        SERVER_READY=true
        break
    fi
    if command -v lsof &> /dev/null; then
        if lsof -ti:"$PORT_API" &> /dev/null; then
            SERVER_READY=true
            break
        fi
    elif command -v netstat &> /dev/null; then
        if netstat -an 2>/dev/null | grep -q ":$PORT_API.*LISTEN"; then
            SERVER_READY=true
            break
        fi
    elif command -v ss &> /dev/null; then
        if ss -lnt 2>/dev/null | grep -q ":$PORT_API"; then
            SERVER_READY=true
            break
        fi
    else
        SERVER_READY=true
        break
    fi
    sleep 0.5
    WAITED=$((WAITED + 1))
done

if [ "$SERVER_READY" != "true" ]; then
    test_failed "HTTP client - server readiness" "Server did not start"
    echo ""
    return 0
fi

response=$(curl -s "http://127.0.0.1:${PORT_API}/api/fetch" || true)

if echo "$response" | grep -q "\"statusCode\":200"; then
    test_passed "HTTP client - GET returns JSON"
else
    test_failed "HTTP client - GET returns JSON" "Unexpected output"
    echo "$response"
    echo "--- kdeps log ---"
    tail -20 "$KDEPS_LOG"
fi

echo ""
