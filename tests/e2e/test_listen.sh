#!/usr/bin/env bash
# E2E: deploy.input.listen. kdeps run follows a file, reads a named pipe (Linux
# and macOS), takes signed webhook POSTs and keeps a websocket open; each event
# runs the workflow once.

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

# Webhook: a signed POST runs the workflow and gets the result back (wait: true).
HDIR=$(mktemp -d)
mkdir -p "$HDIR/home" "$HDIR/data" "$HDIR/proj/resources"
HPORT=$((20000 + RANDOM % 20000))
cat > "$HDIR/proj/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: webhook-e2e
  version: "1.0.0"
  targetActionId: echo
deploy:
  input:
    listen:
      - id: hook
        webhook:
          addr: "127.0.0.1:$HPORT"
          path: /hooks/github
          wait: true
          verify:
            secret: [KDEPS_E2E_HOOK_SECRET]
            signature: '{{trimPrefix (index .headers "X-Hub-Signature-256") "sha256="}}'
YAML
cat > "$HDIR/proj/resources/echo.yaml" <<'YAML'
actionId: echo
name: Echo
apiResponse:
  success: true
  response:
    action: "{{ input('action') }}"
YAML
(cd "$HDIR/proj" && HOME="$HDIR/home" USERPROFILE="$(to_native_path "$HDIR/home")" \
    KDEPS_DATA_DIR="$(to_native_path "$HDIR/data")" KDEPS_E2E_HOOK_SECRET=shh \
    "$KDEPS_BIN" run workflow.yaml > "$HDIR/out.log" 2>&1) &
HPID=$!
BODY='{"action":"opened"}'
SIG=sha256=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac shh | awk '{print $NF}')
REPLY=""
for i in $(seq 1 100); do
    REPLY=$(curl -s -X POST -H 'Content-Type: application/json' -H "X-Hub-Signature-256: $SIG" \
        --data "$BODY" "http://127.0.0.1:$HPORT/hooks/github" 2>/dev/null) && [ -n "$REPLY" ] && break
    sleep 0.2
done
if printf '%s' "$REPLY" | grep -qF '"action":"opened"'; then
    test_passed "listen - a signed webhook POST answers with the run result"
else
    test_failed "listen - signed webhook" "reply=$REPLY $(tail -20 "$HDIR/out.log")"
fi
CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H 'Content-Type: application/json' \
    -H 'X-Hub-Signature-256: sha256=00' --data "$BODY" "http://127.0.0.1:$HPORT/hooks/github")
if [ "$CODE" = "401" ]; then
    test_passed "listen - a webhook POST with a wrong signature gets 401"
else
    test_failed "listen - wrong signature" "status $CODE"
fi
pkill -P "$HPID" 2>/dev/null || true
kill "$HPID" 2>/dev/null || true
wait "$HPID" 2>/dev/null || true
rm -rf "$HDIR"

# Websocket: a fake gateway sends hello; kdeps identifies, sends heartbeats
# with the last sequence, and runs the workflow for MESSAGE_CREATE.
if command -v python3 >/dev/null 2>&1; then
    WDIR=$(mktemp -d)
    mkdir -p "$WDIR/home" "$WDIR/data" "$WDIR/proj/resources"
    WPORT=$((20000 + RANDOM % 20000))
    python3 "$SCRIPT_DIR/helpers/ws_gateway.py" "$WPORT" "$WDIR/gw.log" &
    GWPID=$!
    cat > "$WDIR/proj/workflow.yaml" <<YAML
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: websocket-e2e
  version: "1.0.0"
  targetActionId: echo
deploy:
  input:
    listen:
      - id: gw
        websocket:
          env:
            TOKEN: [KDEPS_E2E_WS_TOKEN]
          url: "ws://127.0.0.1:$WPORT/gateway"
          heartbeat:
            send: '{"op":1,"d":{{or .state.seq "null"}}}'
          on:
            - when: '{{if eq (int .msg.op) 10}}y{{end}}'
              every: '{{.msg.d.heartbeat_interval}}ms'
              send: '{"op":2,"d":{"token":"{{.env.TOKEN}}"}}'
            - when: '{{.msg.s}}'
              set:
                seq: '{{.msg.s}}'
            - when: '{{if eq .msg.t "MESSAGE_CREATE"}}y{{end}}'
              emit: '{{json .msg.d}}'
YAML
    cat > "$WDIR/proj/resources/echo.yaml" <<'YAML'
actionId: echo
name: Echo
apiResponse:
  success: true
  response:
    got: "{{ input('content') }}"
YAML
    (cd "$WDIR/proj" && HOME="$WDIR/home" USERPROFILE="$(to_native_path "$WDIR/home")" \
        KDEPS_DATA_DIR="$(to_native_path "$WDIR/data")" KDEPS_E2E_WS_TOKEN=tok \
        "$KDEPS_BIN" run workflow.yaml > "$WDIR/out.log" 2>&1) &
    WPID=$!
    OK=false
    for i in $(seq 1 100); do
        if grep -qF '[listen gw] {"got":"hello from the gateway"}' "$WDIR/out.log" 2>/dev/null \
            && grep -qF '{"op":1,"d":1}' "$WDIR/gw.log" 2>/dev/null; then
            OK=true
            break
        fi
        sleep 0.2
    done
    if [ "$OK" = true ] && grep -qF '{"op":2,"d":{"token":"tok"}}' "$WDIR/gw.log"; then
        test_passed "listen - a websocket listener identifies, sends heartbeats and runs the workflow per event"
    else
        test_failed "listen - websocket" "$(tail -10 "$WDIR/out.log"; cat "$WDIR/gw.log" 2>/dev/null)"
    fi
    pkill -P "$WPID" 2>/dev/null || true
    kill "$WPID" "$GWPID" 2>/dev/null || true
    wait "$WPID" "$GWPID" 2>/dev/null || true
    rm -rf "$WDIR"
else
    echo "  (skip websocket: python3 not found)"
fi

# A listener asset: telegram_updates without a token logs which variables
# and vault secret it needs, and keeps running.
TDIR=$(mktemp -d)
mkdir -p "$TDIR/home" "$TDIR/data" "$TDIR/proj/resources"
cat > "$TDIR/proj/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: telegram-listener-e2e
  version: "1.0.0"
  targetActionId: reply
deploy:
  input:
    listen:
      - listener: telegram_updates
YAML
cat > "$TDIR/proj/resources/reply.yaml" <<'YAML'
actionId: reply
name: Reply
component:
  name: telegram_post
  with:
    path: sendMessage
    body:
      chat_id: "{{ input('message').chat.id }}"
      text: "pong"
YAML
trun() {
    (cd "$TDIR/proj" && HOME="$TDIR/home" USERPROFILE="$(to_native_path "$TDIR/home")" \
        KDEPS_DATA_DIR="$(to_native_path "$TDIR/data")" TELEGRAM_BOT_TOKEN="" TELEGRAM_TOKEN="" \
        KDEPS_VAULT_PASSWORD="" "$KDEPS_BIN" "$@")
}
OUT=$(trun validate workflow.yaml 2>&1)
if [ $? -eq 0 ]; then
    test_passed "listen - a workflow with listener: telegram_updates and a telegram_post reply validates"
else
    test_failed "listen - telegram validate" "$OUT"
fi
trun run workflow.yaml > "$TDIR/out.log" 2>&1 &
TPID=$!
FOUND=false
for i in $(seq 1 75); do
    if grep -qF "TELEGRAM_BOT_TOKEN or TELEGRAM_TOKEN or the vault secret telegram_bot_token" "$TDIR/out.log" 2>/dev/null; then
        FOUND=true
        break
    fi
    sleep 0.2
done
if [ "$FOUND" = true ] && kill -0 "$TPID" 2>/dev/null; then
    test_passed "listen - telegram_updates without a token names its sources and keeps running"
else
    test_failed "listen - telegram no token" "$(tail -20 "$TDIR/out.log")"
fi
pkill -P "$TPID" 2>/dev/null || true
kill "$TPID" 2>/dev/null || true
wait "$TPID" 2>/dev/null || true
rm -rf "$TDIR"

rm -rf "$LDIR"
echo ""
