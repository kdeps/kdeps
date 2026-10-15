# shellcheck shell=bash
# E2E: the knowledge vault - kdeps knowledge init, build, recall and check.
#   - init scaffolds an existing folder without touching its files and
#     records the vault in config.yaml
#   - build reads notes, wikilinks, frontmatter links and the Go code behind
#     a card's repo link; a second build re-reads nothing
#   - recall names the related notes for a request, within a token budget
#   - check reports facts with more than one home and dangling links
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing kdeps knowledge..."

KN_DIR=$(mktemp -d)
trap 'rm -rf "$KN_DIR"' EXIT
KN_VAULT="$KN_DIR/notes"
KN_REPO="$KN_DIR/repo"
mkdir -p "$KN_VAULT/01 Projects/app" "$KN_REPO/store"
printf 'mine\n' > "$KN_VAULT/KDEPS.md"
printf 'module example.com/app\n\ngo 1.26\n' > "$KN_REPO/go.mod"
printf 'package main\n\nimport _ "example.com/app/store"\n\nfunc main() {}\n' > "$KN_REPO/main.go"
printf 'package store\n' > "$KN_REPO/store/store.go"

# kn runs kdeps with its own config, data dir and home.
kn() {
    HOME="$KN_DIR" KDEPS_CONFIG_PATH="$KN_DIR/config.yaml" KDEPS_DATA_DIR="$KN_DIR/data" "$KDEPS_BIN" knowledge "$@" 2>&1 </dev/null
}

OUT=$(kn init "$KN_VAULT")
if output_grep "created .*/schema.md" "$OUT" && output_grep "Set knowledge.vault" "$OUT" &&
    [ "$(cat "$KN_VAULT/KDEPS.md")" = "mine" ] && [ -d "$KN_VAULT/records/projects" ] &&
    [ ! -d "$KN_VAULT/Projects" ] && grep -q "$KN_VAULT" "$KN_DIR/config.yaml"; then
    test_passed "knowledge - init scaffolds without touching existing files, reuses 01 Projects"
else
    test_failed "knowledge - init" "$OUT"
fi

cat > "$KN_VAULT/records/projects/app.md" <<CARD
---
type: project
status: active
next: ship recall
updated: 2026-10-15
---
| Source | Path |
|---|---|
| repo | $KN_REPO |
| vault | 01 Projects/app |
CARD
cat > "$KN_VAULT/01 Projects/app/Database choice.md" <<'NOTE'
---
type: decision
summary: Postgres over MongoDB for joins and transactions.
tags: [database]
links: ["[[Webhook retries]]"]
---
We picked Postgres.
NOTE
mkdir -p "$KN_VAULT/Resources"
cat > "$KN_VAULT/Resources/Webhook retries.md" <<'NOTE'
---
type: gotcha
summary: Webhooks retry for 3 days; handlers must be idempotent.
---
See [[Nowhere]].
NOTE

OUT=$(kn build)
OUT2=$(kn build)
if output_grep "notes \([0-9]+ changed\), 2 code files" "$OUT" && output_grep "\(0 changed\)" "$OUT2" &&
    [ -f "$KN_DIR/data/notes-"*/knowledge/graph.json ] && [ ! -e "$KN_VAULT/graph.json" ]; then
    test_passed "knowledge - build reads notes and code, keeps the graph outside the vault, reuses its cache"
else
    test_failed "knowledge - build" "$OUT $OUT2"
fi

OUT=$(kn recall "why postgres")
if output_grep "01 Projects/app/Database choice.md" "$OUT" && output_grep "Resources/Webhook retries.md" "$OUT" &&
    output_grep "[0-9]+ notes, ~[0-9]+ tokens" "$OUT"; then
    test_passed "knowledge - recall names the decision and the gotcha it links"
else
    test_failed "knowledge - recall" "$OUT"
fi

printf -- '---\nsummary: x\nnext: do it\n---\n' > "$KN_VAULT/Resources/Stray status.md"
if OUT=$(kn check); then
    test_failed "knowledge - check fails on findings" "$OUT"
elif output_grep "Stray status.md: has a next step; it belongs in the project card" "$OUT" &&
    output_grep "links to \[\[Nowhere\]\]" "$OUT"; then
    test_passed "knowledge - check reports a second home for status and a dangling link"
else
    test_failed "knowledge - check" "$OUT"
fi

OUT=$(kn report)
if output_grep "# Knowledge graph report" "$OUT" && output_grep "## God nodes" "$OUT" &&
    output_grep "## Communities" "$OUT" && output_grep "## Surprising connections" "$OUT" &&
    output_grep "\[\[Nowhere\]\]" "$OUT" && [ -f "$KN_DIR/data/notes-"*/knowledge/graph.html ]; then
    test_passed "knowledge - report lists god nodes, Louvain communities, surprising links; graph.html written"
else
    test_failed "knowledge - report" "$OUT"
fi

# Default vault: no config, no folder argument -> ~/.kdeps/notes.
KD_HOME="$KN_DIR/default-home"
mkdir -p "$KD_HOME"
OUT=$(HOME="$KD_HOME" KDEPS_CONFIG_PATH="$KD_HOME/config.yaml" KDEPS_DATA_DIR="$KD_HOME/data" "$KDEPS_BIN" knowledge init 2>&1 </dev/null)
if [ -f "$KD_HOME/.kdeps/notes/KDEPS.md" ] && [ -d "$KD_HOME/.kdeps/notes/records/projects" ] &&
    ! output_grep "Set knowledge.vault" "$OUT"; then
    test_passed "knowledge - init without a folder sets up the default vault ~/.kdeps/notes"
else
    test_failed "knowledge - default vault" "$OUT"
fi

# --extract with an unreachable model: each note fails and is retried later;
# the build still finishes.
OUT=$(HOME="$KN_DIR" KDEPS_CONFIG_PATH="$KN_DIR/config.yaml" KDEPS_DATA_DIR="$KN_DIR/data" OLLAMA_HOST=http://127.0.0.1:1 \
    "$KDEPS_BIN" knowledge build --extract --model e2e-missing --limit 2 2>&1 </dev/null)
if output_grep "Reading changed notes with e2e-missing" "$OUT" && output_grep "2 notes read \(2 failed\)" "$OUT" &&
    output_grep "notes \([0-9]+ changed\)" "$OUT"; then
    test_passed "knowledge - build --extract records failures and still builds the graph"
else
    test_failed "knowledge - build --extract" "$OUT"
fi
