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

# E2E test for webserver-static example

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing Webserver Static Example..."

WORKFLOW_PATH="$(find_example_dir webserver-static)/workflow.yaml"
[ ! -f "$WORKFLOW_PATH" ] && { test_skipped "Webserver Static (workflow not found)"; return 0; }

PORT=$(grep -E "portNum:\s*[0-9]+" "$WORKFLOW_PATH" | head -1 | sed 's/.*portNum:[[:space:]]*\([0-9]*\).*/\1/' || echo "16395")

if "$KDEPS_BIN" validate "$WORKFLOW_PATH" &> /dev/null; then
    test_passed "Webserver Static - Workflow validation"
else
    test_failed "Webserver Static - Workflow validation" "Validation failed"
    return 0
fi

SERVER_LOG=$(mktemp)
timeout 30 "$KDEPS_BIN" run "$WORKFLOW_PATH" > "$SERVER_LOG" 2>&1 &
SERVER_PID=$!
if ! wait_for_kdeps_port "$PORT" 20; then SERVER_READY=false; else SERVER_READY=true; fi

if [ "$SERVER_READY" = false ]; then
    fail_server_startup "Webserver Static - Server startup (server did not start)" "$SERVER_LOG"
    kill $SERVER_PID 2>/dev/null || true; wait $SERVER_PID 2>/dev/null || true; rm -f "$SERVER_LOG"
    return 0
fi
test_passed "Webserver Static - Server startup"

if command -v curl &> /dev/null; then
    RESP=$(curl -s -w "\n%{http_code}" -X GET "http://127.0.0.1:$PORT/" 2>/dev/null || echo -e "\n000")
    STATUS=$(echo "$RESP" | tail -n 1)
    if [ "$STATUS" = "200" ] || [ "$STATUS" = "301" ] || [ "$STATUS" = "302" ]; then
        test_passed "Webserver Static - GET / (responded with $STATUS)"
    else
        test_failed "Webserver Static - GET / (status $STATUS)"
    fi
fi

kill $SERVER_PID 2>/dev/null || true; wait $SERVER_PID 2>/dev/null || true; rm -f "$SERVER_LOG"
echo ""
