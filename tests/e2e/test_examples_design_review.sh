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

# E2E: the design-review example. Four reviewers check a doc; a rejected doc
# gets a question with numbered options, and the answer request reads the
# findings back from the session.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing design-review example..."

if ! command -v python3 &> /dev/null; then
    test_skipped "Design review - python3 not available"
    return 0 2>/dev/null || exit 0
fi

SRC="$PROJECT_ROOT/examples/design-review"
if "$KDEPS_BIN" validate "$SRC/workflow.yaml" &>/dev/null; then
    test_passed "Design review - workflow validates"
else
    test_failed "Design review - workflow validates" "kdeps validate failed"
fi

API_PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("",0)); print(s.getsockname()[1]); s.close()')
TEST_DIR=$(mktemp -d)
LOG_FILE=$(mktemp)
KDEPS_PID=""
trap 'kill $KDEPS_PID 2>/dev/null; rm -rf "$TEST_DIR" "$LOG_FILE"' EXIT
cp -R "$SRC/." "$TEST_DIR/"
python3 - "$TEST_DIR/workflow.yaml" "$API_PORT" <<'PYEOF'
import sys
p, port = sys.argv[1], sys.argv[2]
s = open(p).read().replace("portNum: 16400", 'hostIp: "127.0.0.1"\n    portNum: ' + port)
open(p, "w").write(s)
PYEOF

"$KDEPS_BIN" run "$TEST_DIR/workflow.yaml" > "$LOG_FILE" 2>&1 &
KDEPS_PID=$!
if ! wait_for_kdeps_port "$API_PORT" 20 "" "$LOG_FILE"; then
    fail_server_startup "Design review - server failed to start" "$LOG_FILE"
    return 0 2>/dev/null || exit 0
fi
API_PORT="$KDEPS_BOUND_PORT"

JAR="$TEST_DIR/cookies"
review() {
    curl -sf --max-time 15 -c "$JAR" -b "$JAR" -H 'Content-Type: application/json' \
        -d "$1" "http://127.0.0.1:${API_PORT}/api/v1/review" 2>&1
}

RESP=$(review '{"doc": "Add a read-through cache in front of the API."}')
if echo "$RESP" | grep -q '"passed":true' && echo "$RESP" | grep -q '"question":""'; then
    test_passed "Design review - a clean doc passes with no question"
else
    test_failed "Design review - a clean doc passes with no question" "resp=$RESP"
fi

RESP=$(review '{"doc": "Retries: TBD"}')
if echo "$RESP" | grep -q '"passed":false' && echo "$RESP" | grep -q 'Revise the design or ship it as is?' \
    && echo "$RESP" | grep -q '2. Ship as is' && echo "$RESP" | grep -q 'TBD left in the doc'; then
    test_passed "Design review - a rejected doc ends with a question and options"
else
    test_failed "Design review - a rejected doc ends with a question and options" "resp=$RESP"
fi

RESP=$(review '{"answer": "2"}')
if echo "$RESP" | grep -q 'Shipping with the open findings above.' && echo "$RESP" | grep -q 'TBD left in the doc'; then
    test_passed "Design review - the answer reads the findings from the session"
else
    test_failed "Design review - the answer reads the findings from the session" "resp=$RESP"
fi
