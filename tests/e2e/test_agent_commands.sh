# shellcheck shell=bash
# E2E: slash commands come from the commands asset set. A command published
# only as an asset (installed with kdeps update) runs its steps in the REPL.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing declared slash commands (commands assets)..."

CMDS_HOME=$(mktemp -d)
CMDS_ROOT="$CMDS_HOME/assets"
mkdir -p "$CMDS_ROOT/commands"
trap 'rm -rf "$CMDS_HOME"' EXIT

# An asset-only command: /e2e-frugal runs two other slash commands.
cat > "$CMDS_ROOT/commands/e2e-frugal.yaml" <<'YAML'
version: 1.0.0
name: e2e-frugal
usage:
  - help: E2E declared command - apply the frugal preset, then list briefing topics
run:
  - command: /efficiency preset frugal
  - command: /instruct list
YAML
CMDS_SHA=$(python3 -c 'import hashlib,sys;print(hashlib.sha256(open(sys.argv[1],"rb").read()).hexdigest())' \
    "$CMDS_ROOT/commands/e2e-frugal.yaml")
cat > "$CMDS_ROOT/lock.json" <<JSON
{"items": {"commands/e2e-frugal": {"version": "1.0.0", "sha256": "$CMDS_SHA", "updatedAt": "2026-10-12T00:00:00Z"}}}
JSON

OUTPUT=$(printf '/help\n/e2e-frugal\n/review-nope\n/quit\n' \
    | HOME="$CMDS_HOME" KDEPS_ASSETS_DIR="$CMDS_ROOT" KDEPS_ASSETS_URL=off timeout 60 "$KDEPS_BIN" 2>&1 || true)

if output_grep_fixed "/review [focus]" "$OUTPUT" && output_grep_fixed "/frugal" "$OUTPUT"; then
    test_passed "commands - /help lists the shipped declared commands"
else
    test_failed "commands - /help missing declared commands" "Output: $OUTPUT"
fi

if output_grep_fixed "E2E declared command" "$OUTPUT"; then
    test_passed "commands - /help lists an asset-only command"
else
    test_failed "commands - asset-only command missing from /help" "Output: $OUTPUT"
fi

if output_grep_fixed "Briefing topics:" "$OUTPUT"; then
    test_passed "commands - an asset-only command runs its steps"
else
    test_failed "commands - asset-only command did not run" "Output: $OUTPUT"
fi

if output_grep_fixed "Unknown command: /review-nope" "$OUTPUT"; then
    test_passed "commands - unknown commands are still reported"
else
    test_failed "commands - unknown command not reported" "Output: $OUTPUT"
fi
