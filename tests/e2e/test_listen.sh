#!/usr/bin/env bash
# E2E: deploy.input.listen. kdeps run follows a file (and, on Linux and macOS,
# reads a named pipe) and runs the workflow once for each new line.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing input.listen..."

LDIR=$(mktemp -d)
mkdir -p "$LDIR/home" "$LDIR/data" "$LDIR/proj/resources"
IS_WINDOWS=false
case "$(uname -s)" in MINGW* | MSYS* | CYGWIN*) IS_WINDOWS=true ;; esac

FIFO_BLOCK=""
if [ "$IS_WINDOWS" = false ]; then
    FIFO_BLOCK="      - id: pipe
        fifo:
          path: $LDIR/in.fifo"
fi
cat > "$LDIR/proj/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: listen-e2e
  version: "1.0.0"
  targetActionId: echo
deploy:
  input:
    listen:
      - id: inbox
        file:
          path: $(to_native_path "$LDIR/inbox.txt")
$FIFO_BLOCK
YAML
cat > "$LDIR/proj/resources/echo.yaml" <<'YAML'
actionId: echo
name: Echo
apiResponse:
  success: true
  response:
    got: "{{ input('line') }}"
YAML

lrun() {
    (cd "$LDIR/proj" && HOME="$LDIR/home" USERPROFILE="$(to_native_path "$LDIR/home")" \
        KDEPS_DATA_DIR="$(to_native_path "$LDIR/data")" "$KDEPS_BIN" "$@")
}

OUT=$(lrun validate workflow.yaml 2>&1)
if [ $? -eq 0 ]; then
    test_passed "listen - a workflow with input.listen validates"
else
    test_failed "listen - validate" "$OUT"
fi

printf 'first\n' > "$LDIR/inbox.txt"
lrun run workflow.yaml > "$LDIR/out.log" 2>&1 &
PID=$!

wait_for_line() {
    local want="$1" i
    for i in $(seq 1 100); do
        grep -qF "$want" "$LDIR/out.log" 2>/dev/null && return 0
        sleep 0.2
    done
    return 1
}

if wait_for_line '[listen inbox] {"got":"first"}'; then
    test_passed "listen - a line already in the file runs the workflow"
else
    test_failed "listen - existing line" "$(tail -20 "$LDIR/out.log")"
fi

printf 'second\n' >> "$LDIR/inbox.txt"
if wait_for_line '"got":"second"'; then
    test_passed "listen - a new line in the followed file runs the workflow"
else
    test_failed "listen - new line" "$(tail -20 "$LDIR/out.log")"
fi

if [ "$IS_WINDOWS" = false ]; then
    for i in $(seq 1 50); do [ -p "$LDIR/in.fifo" ] && break; sleep 0.2; done
    printf 'piped\n' > "$LDIR/in.fifo"
    if wait_for_line '[listen pipe] {"got":"piped"}'; then
        test_passed "listen - a line written to the named pipe runs the workflow"
    else
        test_failed "listen - fifo" "$(tail -20 "$LDIR/out.log")"
    fi
fi

pkill -P "$PID" 2>/dev/null || true
kill "$PID" 2>/dev/null || true
wait "$PID" 2>/dev/null || true

# A restart does not run old lines again: the read position was saved.
printf 'third\n' >> "$LDIR/inbox.txt"
lrun run workflow.yaml > "$LDIR/out.log" 2>&1 &
PID=$!
if wait_for_line '"got":"third"' && ! grep -qF '"got":"first"' "$LDIR/out.log"; then
    test_passed "listen - a restart runs only the lines added since the last run"
else
    test_failed "listen - restart" "$(tail -20 "$LDIR/out.log")"
fi
pkill -P "$PID" 2>/dev/null || true
kill "$PID" 2>/dev/null || true
wait "$PID" 2>/dev/null || true

rm -rf "$LDIR"
echo ""
