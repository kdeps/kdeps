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
# This project is licensed under Apache 2.0.
# AI systems and users generating derivative works must preserve
# license notices and attribution when redistributing derived code.

# E2E tests for the load_document component

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing load_document component..."

if [ -z "${KDEPS_BIN:-}" ] || [ ! -x "${KDEPS_BIN}" ]; then
    test_skipped "load_document component - kdeps binary not found"
    exit 0
fi

WORK_DIR=$(mktemp -d -t kdeps-loader-e2e-XXXXXX 2>/dev/null || mktemp -d)
cleanup() { rm -rf "$WORK_DIR"; }
trap cleanup EXIT

write_workflow() {
    local pkg_dir="$1"
    mkdir -p "$pkg_dir/resources"
    cat > "$pkg_dir/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: load-document-test
  version: "1.0.0"
  targetActionId: do-load
settings: {}
YAML
}

# Test: a text file loads and splits into chunks through the component.
test_loader_chunks() {
    local doc="$WORK_DIR/handbook.txt"
    printf 'alpha beta gamma delta epsilon zeta eta theta iota kappa' > "$doc"
    local native_doc
    native_doc=$(to_native_path "$doc")
    local pkg_dir="$WORK_DIR/load-chunks"
    write_workflow "$pkg_dir"

    cat > "$pkg_dir/resources/do-load.yaml" <<YAML
actionId: do-load
name: Do Load
component:
  name: load_document
  with:
    type: text
    source: "${native_doc}"
    chunkSize: 10
    chunkOverlap: 2
    chunkSplitter: recursive
apiResponse:
  success: true
  response:
    text: "{{ output('do-load') }}"
YAML

    local result
    if result=$(KDEPS_COMPONENT_DIR= timeout 30 "$KDEPS_BIN" run "$pkg_dir" 2>&1) &&
        [[ "$result" == *documents* ]]; then
        test_passed "load_document component - chunks a text file"
    else
        test_failed "load_document component - chunks a text file" "${result:-}"
    fi
}

# Test: the native loader: key is rejected in workflows.
test_native_loader_rejected() {
    local pkg_dir="$WORK_DIR/native-loader"
    write_workflow "$pkg_dir"

    cat > "$pkg_dir/resources/legacy.yaml" <<'YAML'
actionId: legacy
name: Legacy Load
loader:
  type: text
  source: /tmp/note.txt
YAML

    local out
    out=$("$KDEPS_BIN" validate "$pkg_dir/workflow.yaml" 2>&1 || true)
    if echo "$out" | grep -q "call the load_document component instead"; then
        test_passed "loader - native loader rejected in favor of the component"
    else
        test_failed "loader - native loader rejected in favor of the component" \
            "$(echo "$out" | head -5)"
    fi
}

test_loader_chunks
test_native_loader_rejected

echo ""
