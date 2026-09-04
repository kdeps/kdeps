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

# Unit-style checks for E2E helpers in common.sh (wait_for_kdeps_port,
# llm_env_blocker, fail_server_startup).

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=common.sh
source "$SCRIPT_DIR/common.sh"

echo "Testing E2E helper functions..."

CODE=$(http_status "http://127.0.0.1:1/")
if [ "$CODE" = "000" ]; then
    test_passed "http_status unreachable host is 000"
else
    test_failed "http_status unreachable host is 000" "got $CODE"
fi
if [ "${#CODE}" -eq 3 ]; then
    test_passed "http_status returns exactly three digits"
else
    test_failed "http_status returns exactly three digits" "got '$CODE'"
fi

TIMEOUT_START=$(date +%s)
timeout 8 true
TIMEOUT_ELAPSED=$(( $(date +%s) - TIMEOUT_START ))
if [ "$TIMEOUT_ELAPSED" -le 2 ]; then
    test_passed "timeout returns as soon as the command exits (${TIMEOUT_ELAPSED}s)"
else
    test_failed "timeout returns as soon as the command exits" "took ${TIMEOUT_ELAPSED}s, shim waited out the sleep"
fi

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "wait_for_kdeps_port (python3 not available)"
else
    HELPER_PORT=$(python3 -c "
import socket
s = socket.socket()
s.bind(('127.0.0.1', 0))
print(s.getsockname()[1])
s.close()
")
    python3 - <<PY &
from http.server import BaseHTTPRequestHandler, HTTPServer
class H(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'ok')
    def log_message(self, *a):
        pass
HTTPServer(('127.0.0.1', int('$HELPER_PORT')), H).serve_forever()
PY
    HELPER_PID=$!
    if wait_for_kdeps_port "$HELPER_PORT" 10; then
        test_passed "wait_for_kdeps_port sees a listening HTTP server"
    else
        test_failed "wait_for_kdeps_port sees a listening HTTP server" "port $HELPER_PORT never answered"
    fi
    kill "$HELPER_PID" 2>/dev/null || true
    wait "$HELPER_PID" 2>/dev/null || true
fi

if llm_env_blocker "Error: cannot connect to ollama at 127.0.0.1:11434: connection refused"; then
    test_passed "llm_env_blocker matches ollama connection refused"
else
    test_failed "llm_env_blocker matches ollama connection refused"
fi

if llm_env_blocker "LLM backend setup failed: failed to start ollama: ollama not found in PATH: exec: \"ollama\": executable file not found in %PATH%"; then
    test_passed "llm_env_blocker matches ollama missing from PATH"
else
    test_failed "llm_env_blocker matches ollama missing from PATH"
fi

if llm_env_blocker "panic: runtime error: invalid memory address or nil pointer dereference"; then
    test_failed "llm_env_blocker ignores product panics" "panic was treated as an LLM env skip"
else
    test_passed "llm_env_blocker ignores product panics"
fi

if llm_server_crashed "signal: segmentation fault (core dumped)"; then
    test_passed "llm_server_crashed matches segfault"
else
    test_failed "llm_server_crashed matches segfault"
fi

BLOCK_LOG=$(mktemp)
echo "dial tcp 127.0.0.1:11434: connect: connection refused" > "$BLOCK_LOG"
SKIP_BEFORE=$SKIPPED
FAIL_BEFORE=$FAILED
fail_server_startup "helper-llm-skip" "$BLOCK_LOG" >/dev/null
SKIP_AFTER=$SKIPPED
FAIL_AFTER=$FAILED
SKIPPED=$SKIP_BEFORE
FAILED=$FAIL_BEFORE
export SKIPPED FAILED
if [ "$SKIP_AFTER" -eq $((SKIP_BEFORE + 1)) ] && [ "$FAIL_AFTER" -eq "$FAIL_BEFORE" ]; then
    test_passed "fail_server_startup skips when LLM backend is missing"
else
    test_failed "fail_server_startup skips when LLM backend is missing" "SKIPPED $SKIP_BEFORE->$SKIP_AFTER FAILED $FAIL_BEFORE->$FAIL_AFTER"
fi
rm -f "$BLOCK_LOG"

CRASH_LOG=$(mktemp)
echo "panic: bind: address already in use" > "$CRASH_LOG"
SKIP_BEFORE=$SKIPPED
FAIL_BEFORE=$FAILED
fail_server_startup "helper-product-fail" "$CRASH_LOG" >/dev/null
SKIP_AFTER=$SKIPPED
FAIL_AFTER=$FAILED
SKIPPED=$SKIP_BEFORE
FAILED=$FAIL_BEFORE
export SKIPPED FAILED
if [ "$FAIL_AFTER" -eq $((FAIL_BEFORE + 1)) ] && [ "$SKIP_AFTER" -eq "$SKIP_BEFORE" ]; then
    test_passed "fail_server_startup fails on product crash"
else
    test_failed "fail_server_startup fails on product crash" "SKIPPED $SKIP_BEFORE->$SKIP_AFTER FAILED $FAIL_BEFORE->$FAIL_AFTER"
fi
rm -f "$CRASH_LOG"

echo ""
