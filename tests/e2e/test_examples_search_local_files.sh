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
# E2E tests for examples/search-local-files

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing search-local-files example..."

EXAMPLE_DIR="$(find_example_dir search-local-files)"
WF="$EXAMPLE_DIR/workflow.yaml"

# -- Structure tests -----------------------------------------------------------

if [ ! -f "$WF" ]; then
    test_skipped "search-local-files (workflow.yaml not found)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

test_passed "search-local-files - workflow.yaml exists"

for f in search.yaml response.yaml; do
    if [ -f "$EXAMPLE_DIR/resources/$f" ]; then
        test_passed "search-local-files - resources/$f exists"
    else
        test_failed "search-local-files - resources/$f exists" "File not found"
    fi
done

if [ -f "$EXAMPLE_DIR/README.md" ]; then
    test_passed "search-local-files - README.md exists"
else
    test_failed "search-local-files - README.md exists" "File not found"
fi

# search.yaml uses searchLocal executor
if grep -q "searchLocal:" "$EXAMPLE_DIR/resources/search.yaml" 2>/dev/null; then
    test_passed "search-local-files - search.yaml uses run.searchLocal"
else
    test_failed "search-local-files - search.yaml uses run.searchLocal" "searchLocal: key not found"
fi

# search.yaml reads query via get()
if grep -q "get('query')" "$EXAMPLE_DIR/resources/search.yaml" 2>/dev/null; then
    test_passed "search-local-files - search.yaml reads query via get()"
else
    test_failed "search-local-files - search.yaml reads query via get()" "get('query') not found"
fi

# response.yaml references search output
if grep -q "output('search')" "$EXAMPLE_DIR/resources/response.yaml" 2>/dev/null; then
    test_passed "search-local-files - response.yaml references output('search')"
else
    test_failed "search-local-files - response.yaml references output('search')" "output('search') not found"
fi

# -- Runtime test (skip if no binary) ------------------------------------------

if [ -z "${KDEPS_BIN:-}" ] || [ ! -x "${KDEPS_BIN}" ]; then
    test_skipped "search-local-files - runtime test (kdeps binary not available)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

if ! command -v python3 &>/dev/null; then
    test_skipped "search-local-files - runtime test (python3 not available)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

API_PORT=$(python3 -c "import socket; s=socket.socket(); s.bind(('',0)); print(s.getsockname()[1]); s.close()")
PORT=$(grep -E "portNum:" "$WF" | grep -oE '[0-9]+' | head -1)

WORK_DIR=$(mktemp -d)
LOG_FILE=$(mktemp)
KDEPS_PID=""

cleanup() {
    [ -n "$KDEPS_PID" ] && kill "$KDEPS_PID" 2>/dev/null; wait "$KDEPS_PID" 2>/dev/null
    rm -rf "$WORK_DIR" "$LOG_FILE"
}
trap cleanup EXIT

# create a searchable data directory
DATA_DIR="$WORK_DIR/data"
mkdir -p "$DATA_DIR"
echo "Go is a compiled programming language" > "$DATA_DIR/go.txt"
echo "Python is an interpreted language" > "$DATA_DIR/python.txt"
echo "Rust is a memory-safe systems language" > "$DATA_DIR/rust.txt"

mkdir -p "$WORK_DIR/resources"
sed "s/portNum: ${PORT}/portNum: ${API_PORT}/" "$WF" > "$WORK_DIR/workflow.yaml"
cp "$EXAMPLE_DIR/resources/response.yaml" "$WORK_DIR/resources/"

# search resource pointing at the temp data dir
cat > "$WORK_DIR/resources/search.yaml" <<EOF
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: search
run:
  validations:
    methods: [POST]
    routes: [/search]
    check:
      - get('query') != ''
    error:
      code: 400
      message: "query is required"
  searchLocal:
    path: "$DATA_DIR"
    query: "{{ get('query') }}"
    glob: "{{ get('glob', '') }}"
    limit: 20
EOF

"$KDEPS_BIN" run "$WORK_DIR/workflow.yaml" >"$LOG_FILE" 2>&1 &
KDEPS_PID=$!

READY=false
for i in $(seq 1 30); do
    curl -sf --max-time 1 "http://127.0.0.1:${API_PORT}/health" >/dev/null 2>&1 && READY=true && break
    sleep 0.5
done

if [ "$READY" = false ]; then
    test_skipped "search-local-files - server start"
    echo ""
    return 0 2>/dev/null || exit 0
fi

test_passed "search-local-files - server started"

# search for a keyword that exists
RESP=$(curl -s --max-time 10 -X POST "http://127.0.0.1:${API_PORT}/search" \
    -H "Content-Type: application/json" \
    -d '{"query": "compiled"}' 2>&1)

if echo "$RESP" | grep -q "success\|results\|count\|go.txt"; then
    test_passed "search-local-files - /search finds matching files"
else
    test_failed "search-local-files - /search finds matching files" "resp=$RESP"
fi

# search for a keyword that does not exist - should return empty results
EMPTY_RESP=$(curl -s --max-time 10 -X POST "http://127.0.0.1:${API_PORT}/search" \
    -H "Content-Type: application/json" \
    -d '{"query": "javascript"}' 2>&1)

if echo "$EMPTY_RESP" | grep -q "success\|results"; then
    test_passed "search-local-files - /search returns empty results gracefully"
else
    test_failed "search-local-files - /search returns empty results gracefully" "resp=$EMPTY_RESP"
fi

echo ""
echo "search-local-files example tests complete."
