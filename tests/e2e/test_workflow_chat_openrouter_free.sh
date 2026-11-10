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

# E2E: chat.refine rewrites a workflow chat's prompt with the refine-system
# harness before the real call, which then answers the rewrite. A chat

# E2E: a workflow chat on an OpenRouter free id that the provider rate-limits
# (HTTP 429, "Provider returned error") moves to the next free id, as agent
# mode does. A stand-in OpenRouter server answers every id but the first.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing workflow chat OpenRouter free-model hop..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Workflow chat free-model hop - python3 not available"
    return 0 2>/dev/null || exit 0
fi

IDS=$(grep -E '^  - id: ".*:free"' "$PROJECT_ROOT/assets/models/openrouter.yaml" | head -2 | sed -E 's/.*"(.*)"/\1/')
FIRST=$(echo "$IDS" | sed -n 1p)
SECOND=$(echo "$IDS" | sed -n 2p)

TEST_DIR=$(mktemp -d)
mkdir -p "$TEST_DIR/resources"
LOG_FILE=$(mktemp)
KDEPS_PID=""
OR_PID=""
trap 'kill $KDEPS_PID $OR_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT

OR_PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
FIRST="$FIRST" python3 - "$OR_PORT" <<'PY' &
import json, os, sys
from http.server import BaseHTTPRequestHandler, HTTPServer
first = os.environ["FIRST"]
class H(BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))) or b"{}")
        model = body.get("model", "")
        if model == first:
            data = json.dumps({"error": {"message": "Provider returned error", "code": 429}}).encode()
            self.send_response(429)
        else:
            data = json.dumps({"id": "x", "object": "chat.completion", "model": model, "choices": [
                {"index": 0, "finish_reason": "stop",
                 "message": {"role": "assistant", "content": "ANSWERED BY " + model}}]}).encode()
            self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)
HTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
PY
OR_PID=$!

PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
cat > "$TEST_DIR/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: chat-free-hop-e2e
  version: "1.0.0"
  targetActionId: response
deploy:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: ${PORT}
    routes:
      - path: /ask
        methods: [GET]
YAML

cat > "$TEST_DIR/resources/ask.yaml" <<YAML
actionId: ask
name: Ask
validations:
  routes: [/ask]
chat:
  model:
    backend: openrouter
    name: "${FIRST}"
  context:
    prompt: "hi"
YAML

cat > "$TEST_DIR/resources/response.yaml" <<'YAML'
actionId: response
name: Response
flow:
  requires: [ask]
apiResponse:
  success: true
  response:
    answer: "{{ output('ask') }}"
YAML

KDEPS_LLM_OVERRIDE_URL="" KDEPS_LLM_BASE_URL="http://127.0.0.1:${OR_PORT}" OPENROUTER_API_KEY=e2e \
    "$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Workflow chat free-model hop: server failed to start" "$LOG_FILE"
    return 1 2>/dev/null || exit 1
fi
RESP=$(curl -s --max-time 60 "http://127.0.0.1:${KDEPS_BOUND_PORT}/ask" 2>&1)
kill "$KDEPS_PID" 2>/dev/null
wait "$KDEPS_PID" 2>/dev/null

if [ -n "$SECOND" ] && echo "$RESP" | grep -qF "ANSWERED BY ${SECOND}"; then
    test_passed "Workflow chat free-model hop - a provider 429 moves to the next free id"
else
    test_failed "Workflow chat free-model hop - a provider 429 moves to the next free id" "resp=$RESP $(grep -iE 'error' "$LOG_FILE" | head -3)"
fi
