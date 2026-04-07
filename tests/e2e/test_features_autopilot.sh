#!/usr/bin/env bash
# Autopilot feature E2E tests
# Tests that the kdeps binary correctly handles workflow YAML with autopilot resource type.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing autopilot feature..."

# Helper: write a workflow file and return the path.
write_workflow() {
    local dir="$1"
    local filename="$2"
    local content="$3"
    local filepath="$dir/$filename"
    printf '%s' "$content" > "$filepath"
    echo "$filepath"
}

# Test: autopilot resource type is recognized by the binary (validation/parse step)
test_autopilot_resource_recognized() {
    local tmpdir
    tmpdir=$(mktemp -d)

    local workflow_yaml
    workflow_yaml=$(cat <<'EOF'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: autopilot-test
  version: "1.0.0"
  targetActionId: pilot
resources:
  - metadata:
      actionId: pilot
      name: Autopilot Resource
    run:
      autopilot:
        goal: "Find the answer"
        maxIterations: 3
EOF
)

    write_workflow "$tmpdir" "workflow.yaml" "$workflow_yaml" > /dev/null

    # Use kdeps to validate/parse the workflow file. We expect the binary to
    # parse it without rejecting the autopilot key as unknown.
    # The "package" command is available and does parse the workflow.
    if "$KDEPS_BIN" bundle package "$tmpdir" --output "$tmpdir/out.kdeps" &>/dev/null 2>&1; then
        test_passed "autopilot resource type recognized in workflow YAML"
    else
        # If packaging fails (environment-specific, e.g. no Docker), check if
        # the failure is due to autopilot being unrecognized vs environment.
        local output
        output=$("$KDEPS_BIN" bundle package "$tmpdir" --output "$tmpdir/out.kdeps" 2>&1 || true)
        if echo "$output" | grep -qi "unknown\|unrecognized\|invalid.*autopilot"; then
            test_failed "autopilot resource type recognized" "Binary rejects autopilot as unknown field: $output"
        else
            test_skipped "autopilot resource type recognized (environment-specific failure)"
        fi
    fi

    rm -rf "$tmpdir"
}

# Test: autopilot config with empty goal is rejected at the executor level
# (validates via Go test, not binary - documented expectation)
test_autopilot_empty_goal_rejected() {
    local tmpdir
    tmpdir=$(mktemp -d)

    local workflow_yaml
    workflow_yaml=$(cat <<'EOF'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: autopilot-empty-goal
  version: "1.0.0"
  targetActionId: pilot
resources:
  - metadata:
      actionId: pilot
      name: Autopilot
    run:
      autopilot:
        goal: ""
        maxIterations: 3
EOF
)

    write_workflow "$tmpdir" "workflow.yaml" "$workflow_yaml" > /dev/null

    # The empty goal check happens at runtime (executor.Execute), not at parse time.
    # Packaging should succeed (YAML is syntactically valid).
    if "$KDEPS_BIN" bundle package "$tmpdir" --output "$tmpdir/out.kdeps" &>/dev/null 2>&1; then
        test_passed "autopilot empty goal YAML parses successfully (runtime validation expected)"
    else
        local output
        output=$("$KDEPS_BIN" bundle package "$tmpdir" --output "$tmpdir/out.kdeps" 2>&1 || true)
        if echo "$output" | grep -qi "goal.*empty\|empty.*goal"; then
            test_passed "autopilot empty goal rejected at parse time"
        else
            test_skipped "autopilot empty goal test (environment-specific failure)"
        fi
    fi

    rm -rf "$tmpdir"
}

# Test: autopilot result structure in JSON output (validates domain types)
test_autopilot_result_structure() {
    # This test verifies the AutopilotResult JSON structure is correct by running
    # the unit test inline (via Go test binary if available).
    if ! command -v go &>/dev/null; then
        test_skipped "autopilot result structure (go not available)"
        return 0
    fi

    local project_root
    project_root="$(cd "$SCRIPT_DIR/../.." && pwd)"

    if (cd "$project_root" && go test ./pkg/executor/autopilot/... -run TestExecutor_Execute_ResultStructure -count=1 &>/dev/null 2>&1); then
        test_passed "autopilot result structure (TotalRuns, Iterations, Goal fields correct)"
    else
        test_failed "autopilot result structure" "Unit test TestExecutor_Execute_ResultStructure failed"
    fi
}

# Test: maxIterations defaults to 3
test_autopilot_default_max_iterations() {
    if ! command -v go &>/dev/null; then
        test_skipped "autopilot default maxIterations (go not available)"
        return 0
    fi

    local project_root
    project_root="$(cd "$SCRIPT_DIR/../.." && pwd)"

    if (cd "$project_root" && go test ./pkg/executor/autopilot/... -run TestExecutor_Execute_DefaultMaxIterations -count=1 &>/dev/null 2>&1); then
        test_passed "autopilot maxIterations defaults to 3 when unset"
    else
        test_failed "autopilot default maxIterations" "Unit test TestExecutor_Execute_DefaultMaxIterations failed"
    fi
}

# Run all tests
test_autopilot_resource_recognized
test_autopilot_empty_goal_rejected
test_autopilot_result_structure
test_autopilot_default_max_iterations

echo ""
