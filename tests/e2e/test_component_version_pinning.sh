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

# E2E tests for component version pinning (run.component.version).

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo ""
echo "Testing component version pinning..."

KDEPS_BIN="${KDEPS_BIN:-kdeps}"

TEST_DIR=$(mktemp -d)
cleanup() { rm -rf "$TEST_DIR"; }
trap cleanup EXIT

# ── Setup: create a component with version metadata ────────────────────────

COMP_DIR="$TEST_DIR/workflow/components/echoer"
mkdir -p "$COMP_DIR"

cat > "$COMP_DIR/component.yaml" << 'EOF'
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: echoer
  version: 1.2.0
  description: A simple echo component for testing version pinning
resources:
  - name: echo-res
    actionId: echoEval
    expr:
      - set("msg", "Execution complete")
EOF

# ── Setup: create a workflow that calls the component ──────────────────────

WORKFLOW_DIR="$TEST_DIR/workflow"
mkdir -p "$WORKFLOW_DIR"

write_workflow() {
    local version_field="$1"
    cat > "$WORKFLOW_DIR/workflow.yaml" << EOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: version-test
  version: "1.0.0"
  targetActionId: call-echo
settings: {}
resources:
  - name: echo-caller
    actionId: call-echo
    component:
      name: echoer
$version_field
EOF
}

# ── Test 1: version match ──────────────────────────────────────────────────

write_workflow "      version: 1.2.0"

OUTPUT=$(cd "$TEST_DIR" && KDEPS_SKIP_BOOTSTRAP=1 "$KDEPS_BIN" run workflow/workflow.yaml 2>&1) || true
if echo "$OUTPUT" | grep -q "Execution complete"; then
    test_passed "[PASS] version match — pinned 1.2.0 matches component 1.2.0"
else
    test_failed "[FAIL] version match — pinned 1.2.0 matches component 1.2.0" "got: $(echo "$OUTPUT" | grep -i 'error\|fail\|version\|echo' | tail -3)"
fi

# ── Test 2: version mismatch ───────────────────────────────────────────────

write_workflow "      version: 2.0.0"

OUTPUT=$(cd "$TEST_DIR" && KDEPS_SKIP_BOOTSTRAP=1 "$KDEPS_BIN" run workflow/workflow.yaml 2>&1) || true
if echo "$OUTPUT" | grep -q "version mismatch"; then
    test_passed "[PASS] version mismatch — pinned 2.0.0 rejected, component is 1.2.0"
else
    test_failed "[FAIL] version mismatch — pinned 2.0.0 rejected, component is 1.2.0" "got: $(echo "$OUTPUT" | grep -i 'error\|fail\|version' | tail -3)"
fi

# ── Test 3: no version pinned (backwards compatible) ───────────────────────

write_workflow ""

OUTPUT=$(cd "$TEST_DIR" && KDEPS_SKIP_BOOTSTRAP=1 "$KDEPS_BIN" run workflow/workflow.yaml 2>&1) || true
if echo "$OUTPUT" | grep -q "Execution complete"; then
    test_passed "[PASS] no version pin — runs any component version"
else
    test_failed "[FAIL] no version pin — runs any component version" "got: $(echo "$OUTPUT" | grep -i 'error\|fail\|Execution' | tail -3)"
fi

# ── Test 4: version pinned but component has no version ────────────────────

COMP2_DIR="$TEST_DIR/workflow/components/no-version"
mkdir -p "$COMP2_DIR"

cat > "$COMP2_DIR/component.yaml" << 'EOF'
apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: no-version
  description: Component without version metadata
resources:
  - name: nv-res
    actionId: nvEval
    expr:
      - set("msg", "no-version-output")
EOF

cat > "$WORKFLOW_DIR/workflow.yaml" << EOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: version-test
  version: "1.0.0"
  targetActionId: call-noversion
settings: {}
resources:
  - name: nv-caller
    actionId: call-noversion
    component:
      name: no-version
      version: 1.0.0
EOF

OUTPUT=$(cd "$TEST_DIR" && KDEPS_SKIP_BOOTSTRAP=1 "$KDEPS_BIN" run workflow/workflow.yaml 2>&1) || true
if echo "$OUTPUT" | grep -q "Execution complete"; then
    test_passed "[PASS] version pinned on unversioned component — runs with warning"
else
    test_failed "[FAIL] version pinned on unversioned component — runs with warning" "got: $(echo "$OUTPUT" | grep -i 'error\|fail\|no-version' | tail -3)"
fi

# ── Test 5: validate CLI accepts version field ─────────────────────────────

cat > "$WORKFLOW_DIR/workflow.yaml" << EOF
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: version-test
  version: "1.0.0"
  targetActionId: call-echo
settings: {}
resources:
  - name: echo-caller
    actionId: call-echo
    component:
      name: echoer
      version: 1.2.0
EOF

OUTPUT=$(cd "$TEST_DIR" && KDEPS_SKIP_BOOTSTRAP=1 "$KDEPS_BIN" validate workflow/workflow.yaml 2>&1) || true
if echo "$OUTPUT" | grep -q "Validation successful"; then
    test_passed "[PASS] validate accepts component version field"
else
    test_failed "[FAIL] validate accepts component version field" "got: $(echo "$OUTPUT" | tail -5)"
fi

echo ""
echo "Component version pinning E2E: done"
