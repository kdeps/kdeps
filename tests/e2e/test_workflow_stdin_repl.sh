# shellcheck shell=bash
# E2E: a workflow with settings.llm.executionType: stdin is an interactive
# REPL under kdeps run: each stdin line runs the target with it as the
# message, the result prints as JSON (not a Go map dump), and the workflow is
# not also run once in the background with no message.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing the workflow stdin REPL..."

SR_DIR=$(mktemp -d)
trap 'rm -rf "$SR_DIR"' EXIT
mkdir -p "$SR_DIR/resources"
cat > "$SR_DIR/workflow.yaml" << 'EOF'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: stdin-repl
  version: "1.0.0"
  targetActionId: echo
settings:
  llm:
    executionType: stdin   # kdeps run reads messages from stdin
    prompt: "You: "
EOF
cat > "$SR_DIR/resources/echo.yaml" << 'EOF'
actionId: echo
name: Echo
component:
  name: exec
  with:
    command: echo "got {{ input('message') }}"
EOF

OUT=$(cd "$SR_DIR" && printf 'first line\nsecond line\n/quit\n' | timeout 120 "$KDEPS_BIN" run workflow.yaml 2>&1 || true)

if output_grep_fixed '"stdout": "got first line' "$OUT" && output_grep_fixed '"stdout": "got second line' "$OUT"; then
    test_passed "stdin REPL - each line runs the target and prints JSON"
else
    test_failed "stdin REPL - lines" "$(printf '%s' "$OUT" | tail -c 800)"
fi

if output_grep_fixed '"stdout": "got \n"' "$OUT" || output_grep_fixed "running in background" "$OUT" \
    || output_grep_fixed "map[" "$OUT"; then
    test_failed "stdin REPL - no background run, no map dump" "$(printf '%s' "$OUT" | tail -c 800)"
else
    test_passed "stdin REPL - no empty background run and no Go map dump"
fi

rm -rf "$SR_DIR"
trap - EXIT
