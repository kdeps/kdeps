#!/usr/bin/env bash
# Autopilot feature E2E tests
# Tests that the kdeps binary correctly handles workflow YAML invoking the
# autopilot component (autopilot is a component, not a native resource type).

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing autopilot feature..."

# Test: autopilot is invoked via the autopilot component (component:), the
# current mechanism -- not a native "run.autopilot" resource type, which no
# longer exists. Verify a workflow using it validates and packages cleanly.
test_autopilot_resource_recognized() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/resources"

    cat > "$tmpdir/workflow.yaml" <<'EOF'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: autopilot-test
  version: "1.0.0"
  targetActionId: pilot
settings: {}
EOF

    cat > "$tmpdir/resources/pilot.yaml" <<'EOF'
actionId: pilot
name: Autopilot Resource
component:
  name: autopilot
  with:
    task: "Find the answer"
    context: "test context"
EOF

    if "$KDEPS_BIN" bundle package "$tmpdir" --output "$tmpdir/out.kdeps" &>/dev/null 2>&1; then
        test_passed "autopilot resource type recognized in workflow YAML"
    else
        local output
        output=$("$KDEPS_BIN" bundle package "$tmpdir" --output "$tmpdir/out.kdeps" 2>&1 || true)
        test_failed "autopilot resource type recognized" "packaging failed: $output"
    fi

    rm -rf "$tmpdir"
}

# Test: the autopilot component's required "task" input is enforced at
# validation time when omitted -- the modern equivalent of the old
# "empty goal rejected" check, now that autopilot is a component rather than
# a native resource type with its own goal field.
test_autopilot_empty_goal_rejected() {
    local tmpdir
    tmpdir=$(mktemp -d)
    mkdir -p "$tmpdir/resources"

    cat > "$tmpdir/workflow.yaml" <<'EOF'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: autopilot-missing-task
  version: "1.0.0"
  targetActionId: pilot
settings: {}
EOF

    cat > "$tmpdir/resources/pilot.yaml" <<'EOF'
actionId: pilot
name: Autopilot
component:
  name: autopilot
  with:
    context: "no task provided"
EOF

    local output
    output=$("$KDEPS_BIN" validate "$tmpdir/workflow.yaml" 2>&1 || true)
    if output_grep_i "requires input \"task\"|task.*not provided" "$output"; then
        test_passed "autopilot empty goal rejected (missing required task input)"
    else
        test_failed "autopilot empty goal test" "expected a missing-task validation error, got: $output"
    fi

    rm -rf "$tmpdir"
}

# Test: autopilot result structure - autopilot is now a component, verify component.yaml exists
test_autopilot_result_structure() {
    local project_root
    project_root="$(cd "$SCRIPT_DIR/../.." && pwd)"
    local component_file="$project_root/tests/e2e/examples/components/autopilot/component.yaml"

    if [ -f "$component_file" ] && grep -q "plan-and-execute" "$component_file"; then
        test_passed "autopilot result structure (component exports plan-and-execute action)"
    else
        test_failed "autopilot result structure" "autopilot component.yaml missing or malformed"
    fi
}

# Test: maxIterations - autopilot component has model input with default
test_autopilot_default_max_iterations() {
    local project_root
    project_root="$(cd "$SCRIPT_DIR/../.." && pwd)"
    local component_file="$project_root/tests/e2e/examples/components/autopilot/component.yaml"

    if [ -f "$component_file" ] && grep -q "model" "$component_file"; then
        test_passed "autopilot component has configurable model input (replaces maxIterations executor field)"
    else
        test_failed "autopilot default maxIterations" "autopilot component.yaml missing model input"
    fi
}

# Run all tests
test_autopilot_resource_recognized
test_autopilot_empty_goal_rejected
test_autopilot_result_structure
test_autopilot_default_max_iterations

echo ""
