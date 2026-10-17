# shellcheck shell=bash
# E2E: session notes point into git instead of copying it. In a git
# repository, an agent session's note records the branch and the commits it
# covers (commits: <start>..<end>), the card names the branch, and no commit
# message is copied into the vault. Against the shared mock LLM.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing session git pointers (mock LLM)..."

if ! command -v git >/dev/null 2>&1; then
    test_skipped "knowledge git - git not installed"
    return 0 2>/dev/null || exit 0
fi

KG_DIR=$(cd "$(mktemp -d)" && pwd -P)
trap '[ -n "${KG_KEEP:-}" ] || rm -rf "$KG_DIR"' EXIT
native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }
KG_HOME="$KG_DIR/home"
V="$KG_DIR/vault"
P="$KG_DIR/billing"
mkdir -p "$KG_HOME/.kdeps" "$P"
printf "knowledge:\n  vault: '%s'\n" "$(native "$V")" > "$KG_HOME/.kdeps/config.yaml"
export GIT_CEILING_DIRECTORIES="$KG_DIR" GIT_AUTHOR_NAME=e2e GIT_AUTHOR_EMAIL=e2e@example.com \
    GIT_COMMITTER_NAME=e2e GIT_COMMITTER_EMAIL=e2e@example.com GIT_CONFIG_GLOBAL="$NULL_PATH"
git -C "$P" init -q -b feat/invoices
git -C "$P" commit -q --allow-empty -m "older-history-subject" # history from before the session
git -C "$P" commit -q --allow-empty -m "start"
START=$(git -C "$P" rev-parse --short=12 HEAD)

kg_session() {
    (cd "$P" && printf '%b' "$1" | env HOME="$KG_HOME" USERPROFILE="$(native "$KG_HOME")" \
        KDEPS_CONFIG_PATH="$KG_HOME/.kdeps/config.yaml" KDEPS_ASSETS_DIR="$KG_HOME/store" \
        timeout 180 "$KDEPS_BIN" --new --backend openai --model fake 2>&1) || true
}
# ranged_note is the session note whose range spans start..end (the commit
# lands in whichever conversation thread was open), or "".
ranged_note() { grep -l "^commits: $1\.\.$2" "$V"/projects/billing/sessions/*.md 2>/dev/null | head -1; }

OUT=$(kg_session 'set up the invoice job\n!git commit -q --allow-empty -m "invoice job"\nnow wire the cron\n/quit\n')
END=$(git -C "$P" rev-parse --short=12 HEAD)
N=$(ranged_note "$START" "$END")
if [ -n "$N" ] && grep -q "^branch: feat/invoices" "$N" && grep -q "git log $START\.\.$END" "$N"; then
    test_passed "knowledge git - the session note records its branch and commit range"
else
    test_failed "knowledge git - session note range" "start=$START end=$END $(grep -h '^commits:' "$V"/projects/billing/sessions/*.md 2>&1) $OUT"
fi
if ! grep -rq "older-history-subject" "$V"; then
    test_passed "knowledge git - git history is not copied into the vault"
else
    test_failed "knowledge git - history copied" "$(grep -rl older-history-subject "$V")"
fi
if grep -q "^| branch | feat/invoices |$" "$V/projects/billing/billing.md"; then
    test_passed "knowledge git - the card names the branch"
else
    test_failed "knowledge git - card branch row" "$(cat "$V/projects/billing/billing.md")"
fi
if [ "$(git -C "$P" log --format=%s "$START..$END")" = "invoice job" ]; then
    test_passed "knowledge git - the recorded range expands to the session's commits in git"
else
    test_failed "knowledge git - range expands" "$(git -C "$P" log --oneline "$START..$END")"
fi

git -C "$P" checkout -q -b feat/next
kg_session 'one more thing\n/quit\n' >/dev/null
if grep -q "^| branch | feat/next |$" "$V/projects/billing/billing.md" &&
    [ "$(grep -c '^| branch |' "$V/projects/billing/billing.md")" = "1" ]; then
    test_passed "knowledge git - the card follows a branch switch, one branch row"
else
    test_failed "knowledge git - branch switch" "$(cat "$V/projects/billing/billing.md")"
fi

NOGIT="$KG_DIR/plain"
mkdir -p "$NOGIT"
(cd "$NOGIT" && printf 'hello\n/quit\n' | env HOME="$KG_HOME" USERPROFILE="$(native "$KG_HOME")" \
    KDEPS_CONFIG_PATH="$KG_HOME/.kdeps/config.yaml" KDEPS_ASSETS_DIR="$KG_HOME/store" \
    timeout 180 "$KDEPS_BIN" --new --backend openai --model fake >/dev/null 2>&1)
PN=$(find "$V/projects/plain" -path '*/sessions/*.md' 2>/dev/null | head -1)
if [ -n "$PN" ] && ! grep -q "^commits:" "$PN" && ! grep -q "^## Git" "$PN"; then
    test_passed "knowledge git - outside a repository the note has no git lines"
else
    test_failed "knowledge git - no-git folder" "$(cat "$PN" 2>/dev/null | head -12)"
fi

unset GIT_CEILING_DIRECTORIES GIT_AUTHOR_NAME GIT_AUTHOR_EMAIL GIT_COMMITTER_NAME GIT_COMMITTER_EMAIL GIT_CONFIG_GLOBAL
[ -n "${KG_KEEP:-}" ] || rm -rf "$KG_DIR"
trap - EXIT

echo ""
echo "Knowledge git E2E: done"
