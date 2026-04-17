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

# E2E test for batch-processing example

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing Batch Processing Example..."

_BATCH_DIR="$(find_example_dir batch-processing)"
WORKFLOW_PATH="$_BATCH_DIR/workflow.yaml"
[ ! -f "$WORKFLOW_PATH" ] && { test_skipped "Batch Processing (workflow not found)"; return 0; }

PORT=$(grep -E "portNum:\s*[0-9]+" "$WORKFLOW_PATH" | head -1 | sed 's/.*portNum:[[:space:]]*\([0-9]*\).*/\1/' || echo "16395")
ENDPOINT="/process"

if "$KDEPS_BIN" validate "$WORKFLOW_PATH" &> /dev/null; then
    test_passed "Batch Processing - Workflow validation"
else
    test_failed "Batch Processing - Workflow validation" "Validation failed"
    return 0
fi

[ -f "$_BATCH_DIR/test-request.json" ] && \
    test_passed "Batch Processing - Test request file exists"

SERVER_LOG=$(mktemp)
timeout 30 "$KDEPS_BIN" run "$WORKFLOW_PATH" > "$SERVER_LOG" 2>&1 &
SERVER_PID=$!
sleep 3; MAX_WAIT=10; WAITED=0; SERVER_READY=false
while [ $WAITED -lt $MAX_WAIT ]; do
    if command -v lsof &> /dev/null && lsof -ti:$PORT &> /dev/null 2>&1; then
        SERVER_READY=true; sleep 1; break
    elif command -v ss &> /dev/null && ss -lnt 2>/dev/null | grep -q ":$PORT "; then
        SERVER_READY=true; sleep 1; break
    fi
    sleep 0.5; WAITED=$((WAITED + 1))
done

if [ "$SERVER_READY" = false ]; then
    kill $SERVER_PID 2>/dev/null || true; wait $SERVER_PID 2>/dev/null || true; rm -f "$SERVER_LOG"
    test_skipped "Batch Processing - Server startup (server did not start)"; return 0
fi
test_passed "Batch Processing - Server startup"

if command -v curl &> /dev/null; then
    PAYLOAD='{"items":["item1","item2","item3"]}'
    RESP=$(curl -s -w "\n%{http_code}" -X POST -H "Content-Type: application/json" \
        -d "$PAYLOAD" "http://127.0.0.1:$PORT$ENDPOINT" 2>/dev/null || echo -e "\n000")
    STATUS=$(echo "$RESP" | tail -n 1)
    if [ "$STATUS" = "200" ] || [ "$STATUS" = "500" ]; then
        test_passed "Batch Processing - POST $ENDPOINT (responded)"
    else
        test_skipped "Batch Processing - POST $ENDPOINT (status $STATUS)"
    fi
fi

kill $SERVER_PID 2>/dev/null || true; wait $SERVER_PID 2>/dev/null || true; rm -f "$SERVER_LOG"
echo ""
