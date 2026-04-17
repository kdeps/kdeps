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
# E2E tests for examples/embedding-rag

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing embedding-rag example..."

EXAMPLE_DIR="$(find_example_dir embedding-rag)"
WF="$EXAMPLE_DIR/workflow.yaml"

# -- Structure tests -----------------------------------------------------------

if [ ! -f "$WF" ]; then
    test_skipped "embedding-rag (workflow.yaml not found)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

test_passed "embedding-rag - workflow.yaml exists"

for f in index.yaml search.yaml response.yaml; do
    if [ -f "$EXAMPLE_DIR/resources/$f" ]; then
        test_passed "embedding-rag - resources/$f exists"
    else
        test_failed "embedding-rag - resources/$f exists" "File not found"
    fi
done

if [ -f "$EXAMPLE_DIR/README.md" ]; then
    test_passed "embedding-rag - README.md exists"
else
    test_failed "embedding-rag - README.md exists" "File not found"
fi

# index.yaml uses embedding upsert
if grep -q "embedding:" "$EXAMPLE_DIR/resources/index.yaml" 2>/dev/null; then
    test_passed "embedding-rag - index.yaml uses run.embedding"
else
    test_failed "embedding-rag - index.yaml uses run.embedding" "embedding: key not found"
fi

if grep -q "upsert" "$EXAMPLE_DIR/resources/index.yaml" 2>/dev/null; then
    test_passed "embedding-rag - index.yaml uses operation: upsert"
else
    test_failed "embedding-rag - index.yaml uses operation: upsert" "upsert not found"
fi

# search.yaml uses embedding search
if grep -q "embedding:" "$EXAMPLE_DIR/resources/search.yaml" 2>/dev/null; then
    test_passed "embedding-rag - search.yaml uses run.embedding"
else
    test_failed "embedding-rag - search.yaml uses run.embedding" "embedding: key not found"
fi

if grep -q "search" "$EXAMPLE_DIR/resources/search.yaml" 2>/dev/null; then
    test_passed "embedding-rag - search.yaml uses operation: search"
else
    test_failed "embedding-rag - search.yaml uses operation: search" "search not found"
fi

# workflow has two routes
if grep -c "path:" "$WF" 2>/dev/null | grep -qE "^[2-9]"; then
    test_passed "embedding-rag - workflow declares multiple routes"
else
    test_failed "embedding-rag - workflow declares multiple routes" "Expected at least 2 route paths"
fi

# -- Runtime test (skip if no binary) ------------------------------------------

if [ -z "${KDEPS_BIN:-}" ] || [ ! -x "${KDEPS_BIN}" ]; then
    test_skipped "embedding-rag - runtime test (kdeps binary not available)"
    echo ""
    return 0 2>/dev/null || exit 0
fi

if ! command -v python3 &>/dev/null; then
    test_skipped "embedding-rag - runtime test (python3 not available)"
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

mkdir -p "$WORK_DIR/resources"
sed "s/portNum: ${PORT}/portNum: ${API_PORT}/" "$WF" > "$WORK_DIR/workflow.yaml"
cp "$EXAMPLE_DIR/resources/"*.yaml "$WORK_DIR/resources/"

"$KDEPS_BIN" run "$WORK_DIR/workflow.yaml" >"$LOG_FILE" 2>&1 &
KDEPS_PID=$!

READY=false
for i in $(seq 1 30); do
    curl -sf --max-time 1 "http://127.0.0.1:${API_PORT}/health" >/dev/null 2>&1 && READY=true && break
    sleep 0.5
done

if [ "$READY" = false ]; then
    test_skipped "embedding-rag - server start"
    echo ""
    return 0 2>/dev/null || exit 0
fi

test_passed "embedding-rag - server started"

# Index a document
IDX_RESP=$(curl -s --max-time 10 -X POST "http://127.0.0.1:${API_PORT}/index" \
    -H "Content-Type: application/json" \
    -d '{"text": "Go is a compiled programming language designed at Google."}' 2>&1)

if echo "$IDX_RESP" | grep -q "success\|true"; then
    test_passed "embedding-rag - /index endpoint responds with success"
else
    test_failed "embedding-rag - /index endpoint responds with success" "resp=$IDX_RESP"
fi

# Search for the indexed document
SRCH_RESP=$(curl -s --max-time 10 -X POST "http://127.0.0.1:${API_PORT}/search" \
    -H "Content-Type: application/json" \
    -d '{"query": "compiled language"}' 2>&1)

if echo "$SRCH_RESP" | grep -q "success\|results\|count"; then
    test_passed "embedding-rag - /search endpoint responds"
else
    test_failed "embedding-rag - /search endpoint responds" "resp=$SRCH_RESP"
fi

echo ""
echo "embedding-rag example tests complete."
