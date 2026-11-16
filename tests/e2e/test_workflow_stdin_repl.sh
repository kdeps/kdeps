# shellcheck shell=bash
# E2E: a workflow with the interactive input source is a prompt under kdeps
# run: each stdin line runs the target with it as the message and the repl
# session id as get('session'), the result prints as JSON (not a Go map
# dump), and the workflow is not also run once in the background with no
# message. The removed repl.executionType fails validation.
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
deploy:
  input:
    sources: [interactive]   # kdeps run reads messages from stdin
repl:
  prompt: "You: "
  sessionId: e2e-sess
EOF
cat > "$SR_DIR/resources/echo.yaml" << 'EOF'
actionId: echo
name: Echo
component:
  name: exec
  with:
    command: echo "got {{ input('message') }} s={{ get('session') }}"
EOF

OUT=$(cd "$SR_DIR" && printf 'first line\nsecond line\n/quit\n' | timeout 120 "$KDEPS_BIN" run workflow.yaml 2>&1 || true)

# Windows' echo keeps the quotes, so match the text anywhere in stdout.
if output_grep '"stdout": .*got first line' "$OUT" && output_grep '"stdout": .*got second line' "$OUT"; then
    test_passed "stdin REPL - each line runs the target and prints JSON"
else
    test_failed "stdin REPL - lines" "$(printf '%s' "$OUT" | tail -c 800)"
fi

if output_grep_fixed "s=e2e-sess" "$OUT"; then
    test_passed "stdin REPL - each line carries the repl session id"
else
    test_failed "stdin REPL - session id" "$(printf '%s' "$OUT" | tail -c 800)"
fi

if output_grep '"stdout": "\\?"?got \\?"?(\\r)?\\n"' "$OUT" || output_grep_fixed "running in background" "$OUT" \
    || output_grep_fixed "map[" "$OUT"; then
    test_failed "stdin REPL - no background run, no map dump" "$(printf '%s' "$OUT" | tail -c 800)"
else
    test_passed "stdin REPL - no empty background run and no Go map dump"
fi

sed -i.bak 's/^  sessionId: e2e-sess$/  executionType: stdin/' "$SR_DIR/workflow.yaml"
if "$KDEPS_BIN" validate "$SR_DIR/workflow.yaml" > /dev/null 2>&1; then
    test_failed "stdin REPL - repl.executionType is rejected" "validate accepted the removed key"
else
    test_passed "stdin REPL - repl.executionType is rejected"
fi

rm -rf "$SR_DIR"
trap - EXIT
