#!/usr/bin/env bash
# Offline appliance smoke test (Linux).
#
# Builds an appliance from the api-service template, prepackages it for the
# host with the chat model inside (--include-models), and runs the binary in a
# network namespace with only loopback and an empty HOME. The chat must answer
# from the bundled model.
#
# Usage: scripts/offline-smoke.sh <kdeps binary>
# Needs: Linux, sudo (for unshare --net), curl, the model in ~/.kdeps/models
# or network access to download it once.
set -euo pipefail

KDEPS_BIN=$(realpath "${1:?usage: offline-smoke.sh <kdeps binary>}")
ROOT=$(cd "$(dirname "$0")/.." && pwd)
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
export KDEPS_ASSETS_SRC="$ROOT/assets"

ARCH=$(uname -m)
case "$ARCH" in
x86_64) TARGET=linux-amd64 ;;
aarch64 | arm64) TARGET=linux-arm64 ;;
*) echo "offline-smoke: unsupported arch $ARCH" >&2 && exit 1 ;;
esac

cd "$WORK"
"$KDEPS_BIN" new appliance --template api-service >/dev/null
cd appliance
"$KDEPS_BIN" bundle package . >/dev/null
"$KDEPS_BIN" bundle prepackage appliance-1.0.0.kdeps --arch "$TARGET" --include-models --output "$WORK/dist"
APP="$WORK/dist/appliance-1.0.0-$TARGET"
ls -la "$APP"

# The appliance runs with no network and no ~/.kdeps: only what is inside it.
cat >"$WORK/run.sh" <<'EOF'
set -u
ip link set lo up
export HOME=$(mktemp -d)
export KDEPS_API_AUTH_TOKEN=offline-smoke
"$1" >"$2" 2>&1 &
pid=$!
for _ in $(seq 1 120); do
  curl -s -o /dev/null http://127.0.0.1:16395/health && break
  sleep 2
done
curl -s -m 600 -X POST http://127.0.0.1:16395/api/v1/chat \
  -H "Authorization: Bearer offline-smoke" -H "Content-Type: application/json" \
  -d '{"q":"Reply with one word: what color is the sky on a clear day?"}'
kill "$pid" 2>/dev/null
EOF
OUT=$(sudo unshare --net bash "$WORK/run.sh" "$APP" "$WORK/run.log")
echo "response: $OUT"
if echo "$OUT" | grep -q '"success":true' && echo "$OUT" | grep -q '"content":"[^"]'; then
  echo "offline-smoke: PASS"
else
  echo "offline-smoke: FAIL" >&2
  tail -40 "$WORK/run.log" >&2
  exit 1
fi
