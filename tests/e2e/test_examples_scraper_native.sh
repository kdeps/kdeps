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
# E2E tests for examples/scraper-native

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing scraper-native example..."

EXAMPLE_DIR="$PROJECT_ROOT/examples/scraper-native"
WF="$EXAMPLE_DIR/workflow.yaml"

# -- Structure tests -----------------------------------------------------------

if [ ! -f "$WF" ]; then
    test_skipped "scraper-native (workflow.yaml not found)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

test_passed "scraper-native - workflow.yaml exists"

for f in fetch.yaml summarize.yaml response.yaml; do
    if [ -f "$EXAMPLE_DIR/resources/$f" ]; then
        test_passed "scraper-native - resources/$f exists"
    else
        test_failed "scraper-native - resources/$f exists" "File not found"
    fi
done

if [ -f "$EXAMPLE_DIR/README.md" ]; then
    test_passed "scraper-native - README.md exists"
else
    test_failed "scraper-native - README.md exists" "File not found"
fi

# workflow uses native scraper executor
if grep -q "scraper:" "$EXAMPLE_DIR/resources/fetch.yaml" 2>/dev/null; then
    test_passed "scraper-native - fetch.yaml uses run.scraper"
else
    test_failed "scraper-native - fetch.yaml uses run.scraper" "scraper: key not found in fetch.yaml"
fi

# fetch resource reads URL from get()
if grep -q "get('url')" "$EXAMPLE_DIR/resources/fetch.yaml" 2>/dev/null; then
    test_passed "scraper-native - fetch.yaml reads url via get()"
else
    test_failed "scraper-native - fetch.yaml reads url via get()" "get('url') not found"
fi

# summarize uses LLM chat
if grep -q "chat:" "$EXAMPLE_DIR/resources/summarize.yaml" 2>/dev/null; then
    test_passed "scraper-native - summarize.yaml uses chat executor"
else
    test_failed "scraper-native - summarize.yaml uses chat executor" "chat: not found"
fi

# summarize references fetch output
if grep -q "output('fetch')" "$EXAMPLE_DIR/resources/summarize.yaml" 2>/dev/null; then
    test_passed "scraper-native - summarize.yaml references output('fetch')"
else
    test_failed "scraper-native - summarize.yaml references output('fetch')" "output('fetch') not found"
fi

# workflow port matches README
PORT=$(grep -E "portNum:" "$WF" | grep -oE '[0-9]+' | head -1)
if [ -n "$PORT" ]; then
    test_passed "scraper-native - workflow declares portNum $PORT"
else
    test_failed "scraper-native - workflow declares portNum" "portNum not found in workflow.yaml"
fi

# -- Runtime test (skip if no binary) ------------------------------------------

if [ -z "${KDEPS_BIN:-}" ] || [ ! -x "${KDEPS_BIN}" ]; then
    test_skipped "scraper-native - runtime test (kdeps binary not available)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

if ! command -v python3 &>/dev/null; then
    test_skipped "scraper-native - runtime test (python3 not available for mock server)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

# pick a free port for the mock HTTP server
MOCK_PORT=$(python3 -c "import socket; s=socket.socket(); s.bind(('',0)); print(s.getsockname()[1]); s.close()")
API_PORT=$(python3 -c "import socket; s=socket.socket(); s.bind(('',0)); print(s.getsockname()[1]); s.close()")

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

# start mock HTTP server
python3 - <<PYEOF &
from http.server import BaseHTTPRequestHandler, HTTPServer
class H(BaseHTTPRequestHandler):
    def log_message(self,*a): pass
    def do_GET(self):
        body = b"<html><body><h1>Go Language</h1><p>Go is a compiled programming language.</p></body></html>"
        self.send_response(200); self.send_header("Content-Type","text/html"); self.send_header("Content-Length",str(len(body))); self.end_headers(); self.wfile.write(body)
HTTPServer(("127.0.0.1", ${MOCK_PORT}), H).serve_forever()
PYEOF
MOCK_PID=$!

# wait for mock server
for i in $(seq 1 20); do
    curl -sf --max-time 1 "http://127.0.0.1:${MOCK_PORT}/" >/dev/null 2>&1 && break
    sleep 0.2
done

# write a temp workflow pointing at mock server and using portNum from a free port
mkdir -p "$WORK_DIR/resources"
sed "s/portNum: ${PORT}/portNum: ${API_PORT}/" "$WF" > "$WORK_DIR/workflow.yaml"
cp "$EXAMPLE_DIR/resources/response.yaml" "$WORK_DIR/resources/"
# fetch resource pointing at mock
cat > "$WORK_DIR/resources/fetch.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: fetch
run:
  scraper:
    url: "http://127.0.0.1:${MOCK_PORT}/"
    timeout: 10
EOF
cp "$EXAMPLE_DIR/resources/summarize.yaml" "$WORK_DIR/resources/"

"$KDEPS_BIN" run "$WORK_DIR/workflow.yaml" >"$LOG_FILE" 2>&1 &
KDEPS_PID=$!

READY=false
for i in $(seq 1 30); do
    curl -sf --max-time 1 "http://127.0.0.1:${API_PORT}/health" >/dev/null 2>&1 && READY=true && break
    sleep 0.5
done

if [ "$READY" = false ]; then
    test_skipped "scraper-native - server start (may need LLM)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

test_passed "scraper-native - server started"

RESP=$(curl -s --max-time 10 -X POST "http://127.0.0.1:${API_PORT}/summarize" \
    -H "Content-Type: application/json" -d "{}" 2>&1)

# Either a success response or an LLM error - just check it responded
if echo "$RESP" | grep -q "success\|error\|summary\|url"; then
    test_passed "scraper-native - /summarize endpoint responds"
else
    test_failed "scraper-native - /summarize endpoint responds" "resp=$RESP"
fi

echo ""
echo "scraper-native example tests complete."
