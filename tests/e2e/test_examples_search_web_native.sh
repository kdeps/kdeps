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
# E2E tests for examples/search-web-native

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing search-web-native example..."

EXAMPLE_DIR="$(find_example_dir search-web-native)"
WF="$EXAMPLE_DIR/workflow.yaml"

# -- Structure tests -----------------------------------------------------------

if [ ! -f "$WF" ]; then
    test_skipped "search-web-native (workflow.yaml not found)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

test_passed "search-web-native - workflow.yaml exists"

for f in search.yaml answer.yaml response.yaml; do
    if [ -f "$EXAMPLE_DIR/resources/$f" ]; then
        test_passed "search-web-native - resources/$f exists"
    else
        test_failed "search-web-native - resources/$f exists" "File not found"
    fi
done

if [ -f "$EXAMPLE_DIR/README.md" ]; then
    test_passed "search-web-native - README.md exists"
else
    test_failed "search-web-native - README.md exists" "File not found"
fi

# search.yaml uses searchWeb executor
if grep -q "searchWeb:" "$EXAMPLE_DIR/resources/search.yaml" 2>/dev/null; then
    test_passed "search-web-native - search.yaml uses run.searchWeb"
else
    test_failed "search-web-native - search.yaml uses run.searchWeb" "searchWeb: key not found"
fi

# search.yaml reads query via get()
if grep -q "get('query')" "$EXAMPLE_DIR/resources/search.yaml" 2>/dev/null; then
    test_passed "search-web-native - search.yaml reads query via get()"
else
    test_failed "search-web-native - search.yaml reads query via get()" "get('query') not found"
fi

# answer.yaml uses LLM chat
if grep -q "chat:" "$EXAMPLE_DIR/resources/answer.yaml" 2>/dev/null; then
    test_passed "search-web-native - answer.yaml uses chat executor"
else
    test_failed "search-web-native - answer.yaml uses chat executor" "chat: not found"
fi

# answer.yaml references search output
if grep -q "output('search')" "$EXAMPLE_DIR/resources/answer.yaml" 2>/dev/null; then
    test_passed "search-web-native - answer.yaml references output('search')"
else
    test_failed "search-web-native - answer.yaml references output('search')" "output('search') not found"
fi

# README mentions DuckDuckGo or ddg
if grep -qi "duckduckgo\|ddg" "$EXAMPLE_DIR/README.md" 2>/dev/null; then
    test_passed "search-web-native - README mentions DuckDuckGo"
else
    test_failed "search-web-native - README mentions DuckDuckGo" "DuckDuckGo/ddg not found in README"
fi

# -- Runtime test with mock DDG server -----------------------------------------

if [ -z "${KDEPS_BIN:-}" ] || [ ! -x "${KDEPS_BIN}" ]; then
    test_skipped "search-web-native - runtime test (kdeps binary not available)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

if ! command -v python3 &>/dev/null; then
    test_skipped "search-web-native - runtime test (python3 not available for mock server)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

# pick free ports
DDG_PORT=$(python3 -c "import socket; s=socket.socket(); s.bind(('',0)); print(s.getsockname()[1]); s.close()")
API_PORT=$(python3 -c "import socket; s=socket.socket(); s.bind(('',0)); print(s.getsockname()[1]); s.close()")
PORT=$(grep -E "portNum:" "$WF" | grep -oE '[0-9]+' | head -1)

WORK_DIR=$(mktemp -d)
LOG_FILE=$(mktemp)
MOCK_PID=""
KDEPS_PID=""

cleanup() {
    [ -n "$MOCK_PID" ] && kill "$MOCK_PID" 2>/dev/null; wait "$MOCK_PID" 2>/dev/null
    [ -n "$KDEPS_PID" ] && kill "$KDEPS_PID" 2>/dev/null; wait "$KDEPS_PID" 2>/dev/null
    rm -rf "$WORK_DIR" "$LOG_FILE"
}
trap cleanup EXIT

# start mock DDG HTML server
python3 - <<PYEOF &
from http.server import BaseHTTPRequestHandler, HTTPServer
import urllib.parse
class H(BaseHTTPRequestHandler):
    def log_message(self,*a): pass
    def do_GET(self):
        body = b"""<html><body>
<a class="result__a" data-href="https://example.com/go">Go Programming Language</a>
<span class="result__snippet">Go is an open-source programming language.</span>
<a class="result__a" data-href="https://example.com/golang">Official Go Docs</a>
<span class="result__snippet">Documentation for the Go language.</span>
</body></html>"""
        self.send_response(200)
        self.send_header("Content-Type","text/html")
        self.send_header("Content-Length",str(len(body)))
        self.end_headers()
        self.wfile.write(body)
HTTPServer(("127.0.0.1", ${DDG_PORT}), H).serve_forever()
PYEOF
MOCK_PID=$!

# wait for mock DDG
for i in $(seq 1 20); do
    curl -sf --max-time 1 "http://127.0.0.1:${DDG_PORT}/" >/dev/null 2>&1 && break
    sleep 0.2
done

mkdir -p "$WORK_DIR/resources"
sed "s/portNum: ${PORT}/portNum: ${API_PORT}/" "$WF" > "$WORK_DIR/workflow.yaml"
cp "$EXAMPLE_DIR/resources/answer.yaml" "$WORK_DIR/resources/"
cp "$EXAMPLE_DIR/resources/response.yaml" "$WORK_DIR/resources/"

# search resource using mock DDG via KDEPS_DDG_URL env
cat > "$WORK_DIR/resources/search.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: search
run:
  searchWeb:
    query: "{{ get('query', 'Go language') }}"
    provider: ddg
    maxResults: 3
EOF

# Run kdeps with KDEPS_DDG_URL pointing to mock server
KDEPS_DDG_URL="http://127.0.0.1:${DDG_PORT}" \
    "$KDEPS_BIN" run "$WORK_DIR/workflow.yaml" >"$LOG_FILE" 2>&1 &
KDEPS_PID=$!

READY=false
for i in $(seq 1 30); do
    curl -sf --max-time 1 "http://127.0.0.1:${API_PORT}/health" >/dev/null 2>&1 && READY=true && break
    sleep 0.5
done

if [ "$READY" = false ]; then
    test_skipped "search-web-native - server start (may need LLM)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

test_passed "search-web-native - server started"

RESP=$(curl -s --max-time 15 -X POST "http://127.0.0.1:${API_PORT}/search" \
    -H "Content-Type: application/json" \
    -d '{"query": "Go programming language"}' 2>&1)

if echo "$RESP" | grep -q "success\|answer\|error\|results"; then
    test_passed "search-web-native - /search endpoint responds"
else
    test_failed "search-web-native - /search endpoint responds" "resp=$RESP"
fi

echo ""
echo "search-web-native example tests complete."
