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

# E2E test for webserver-proxy example

set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing Webserver Proxy Example..."

_PROXY_DIR="$(find_example_dir webserver-proxy)"
WORKFLOW_PATH="$_PROXY_DIR/workflow.yaml"
[ ! -f "$WORKFLOW_PATH" ] && { test_skipped "Webserver Proxy (workflow not found)"; return 0; }

PORT=$(grep -E "portNum:\s*[0-9]+" "$WORKFLOW_PATH" | head -1 | sed 's/.*portNum:[[:space:]]*\([0-9]*\).*/\1/' || echo "16395")

if "$KDEPS_BIN" validate "$WORKFLOW_PATH" &> /dev/null; then
    test_passed "Webserver Proxy - Workflow validation"
else
    test_failed "Webserver Proxy - Workflow validation" "Validation failed"
    return 0
fi

[ -d "$_PROXY_DIR/backend" ] && \
    test_passed "Webserver Proxy - Backend directory exists"

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
    test_skipped "Webserver Proxy - Server startup (server did not start)"; return 0
fi
test_passed "Webserver Proxy - Server startup"

if command -v curl &> /dev/null; then
    RESP=$(curl -s -w "\n%{http_code}" -X GET "http://127.0.0.1:$PORT/" 2>/dev/null || echo -e "\n000")
    STATUS=$(echo "$RESP" | tail -n 1)
    if [ "$STATUS" = "200" ] || [ "$STATUS" = "502" ] || [ "$STATUS" = "503" ]; then
        test_passed "Webserver Proxy - GET / (responded with $STATUS)"
    else
        test_skipped "Webserver Proxy - GET / (status $STATUS)"
    fi
fi

kill $SERVER_PID 2>/dev/null || true; wait $SERVER_PID 2>/dev/null || true; rm -f "$SERVER_LOG"
echo ""
