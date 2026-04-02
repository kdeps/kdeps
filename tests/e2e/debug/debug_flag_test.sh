#!/bin/bash
# E2E test for --debug flag

set -e

KDEPS_BIN="${KDEPS_BIN:-./kdeps}"
TEST_DIR="$(mktemp -d)"
trap "rm -rf $TEST_DIR" EXIT

cat > "$TEST_DIR/workflow.yaml" << 'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: debug-test
  version: "1.0.0"
  targetActionId: done
  resources:
    - resources/test.yaml
YAML

mkdir -p "$TEST_DIR/resources"
cat > "$TEST_DIR/resources/test.yaml" << 'YAML'
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: done
  name: test
run:
  expr:
    - "set('result', 'ok')"
YAML

# Test that --debug flag produces debug output
echo "Testing --debug flag..."
if "$KDEPS_BIN" validate "$TEST_DIR/workflow.yaml" --debug 2>&1 | grep -q "enter:"; then
    echo "✓ Debug output detected"
    exit 0
else
    echo "✗ No debug output found"
    exit 1
fi
