#!/bin/bash
# Copyright 2026 Kdeps, KvK 94834768
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# The daily bug-hunt script: catalog order for models, shuffled run,
# one GitHub issue per capture, and a pull request when the tree changed.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing the daily bug-hunt script..."

HUNT="$PROJECT_ROOT/scripts/bug-hunt.sh"
ROOT="$(mktemp -d)"

cat >"$ROOT/catalog.yaml" <<'EOF'
models:
  - id: "fail/one:free"
    backend: openrouter
  - id: "edit/two:free"
    backend: openrouter
  - id: "clean/three:free"
    backend: openrouter
EOF

MODELS="$(BUG_HUNT_CATALOG="$ROOT/catalog.yaml" bash "$HUNT" models)"
if [ "$MODELS" = "$(printf '%s\n' 'fail/one:free' 'edit/two:free' 'clean/three:free')" ]; then
	test_passed "bug-hunt models lists free catalog ids in order"
else
	test_failed "bug-hunt models lists free catalog ids in order" "$MODELS"
fi

cat >"$ROOT/fake-kdeps" <<'EOF'
#!/bin/bash
model=""
prev=""
for a in "$@"; do
	if [ "$prev" = "--model" ]; then
		model="$a"
	fi
	prev="$a"
done
echo "$model" >>"$BUG_HUNT_FAKE_LOG"
cat >/dev/null
case "$model" in
fail/*) exit 1 ;;
edit/*)
	echo "auto-improve: done"
	echo "auto-improve: issue https://example.test/issues/3"
	echo fixed >>hunted.txt
	exit 0
	;;
clean/*)
	echo "auto-improve: done"
	exit 0
	;;
refused/*)
	echo "auto-improve: no issue (provider-only failures)"
	exit 1
	;;
esac
exit 1
EOF
chmod +x "$ROOT/fake-kdeps"

cat >"$ROOT/gh" <<'EOF'
#!/bin/bash
echo "$*" >>"$GH_ARGS"
if [ "$1" = "pr" ] && [ "$2" = "list" ]; then
	if [ -n "${GH_OPEN_PR:-}" ]; then
		printf '%s\n' '[{"headRefName":"bot/bug-hunt-existing"}]'
	else
		printf '%s\n' '[]'
	fi
	exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "create" ]; then
	echo "https://example.test/pull/9"
	exit 0
fi
if [ "$1" = "issue" ] && [ "$2" = "create" ]; then
	if [ -n "${GH_ISSUE_FAIL:-}" ]; then
		echo "gh down" >&2
		exit 1
	fi
	echo "https://example.test/issues/3"
	exit 0
fi
exit 0
EOF
chmod +x "$ROOT/gh"

init_repo() {
	local dir="$1"
	git init -q "$dir"
	git -C "$dir" config user.email "test@example.com"
	git -C "$dir" config user.name "test"
	echo base >"$dir/README"
	git -C "$dir" add README
	git -C "$dir" commit -q -m init
}

# Missing key exits before any model call.
MISS="$(mktemp)"
if env -u OPENROUTER_API_KEY bash "$HUNT" run >"$MISS" 2>&1; then
	test_failed "bug-hunt run requires OPENROUTER_API_KEY" "exit 0"
else
	if grep -q "OPENROUTER_API_KEY" "$MISS"; then
		test_passed "bug-hunt run requires OPENROUTER_API_KEY"
	else
		test_failed "bug-hunt run requires OPENROUTER_API_KEY" "$(cat "$MISS")"
	fi
fi
rm -f "$MISS"

# First id fails, second finishes and edits. Publish is skipped.
REPO="$(mktemp -d)"
init_repo "$REPO"
FAKE_LOG="$(mktemp)"
OUT="$(mktemp)"
GH_ARGS="$(mktemp)"
(
	cd "$REPO"
	PATH="$ROOT:$PATH" \
		GH_ARGS="$GH_ARGS" \
		OPENROUTER_API_KEY=test \
		BUG_HUNT_CATALOG="$ROOT/catalog.yaml" \
		BUG_HUNT_KDEPS="$ROOT/fake-kdeps" \
		BUG_HUNT_FAKE_LOG="$FAKE_LOG" \
		BUG_HUNT_RESET=1 \
		BUG_HUNT_SKIP_PUBLISH=1 \
		BUG_HUNT_SHUFFLE=0 \
		BUG_HUNT_MAX_MODELS=4 \
		BUG_HUNT_TIMEOUT=30 \
		bash "$HUNT" run >"$OUT" 2>&1
)
if [ $? -eq 0 ] && grep -q "fail/one:free" "$FAKE_LOG" && grep -q "edit/two:free" "$FAKE_LOG" \
	&& [ "$(head -n 1 "$FAKE_LOG")" = "fail/one:free" ] \
	&& [ "$(grep -c '^fixed$' "$REPO/hunted.txt")" -eq 1 ] \
	&& grep -q "bug-hunt: hunt 1/3" "$OUT" \
	&& grep -q "bug-hunt: hunt 3/3" "$OUT" \
	&& grep -q "trying the next free model" "$OUT" \
	&& grep -q "skip publish" "$OUT" \
	&& grep -q "auto-improve: issue https://example.test/issues/3" "$OUT" \
	&& [ "$(grep -c 'issue create' "$GH_ARGS")" -eq 3 ]; then
	test_passed "bug-hunt run tries the next free model and keeps the fix"
else
	test_failed "bug-hunt run tries the next free model and keeps the fix" "$(cat "$OUT"; echo ---; cat "$FAKE_LOG"; echo ---; cat "$GH_ARGS")"
fi

# A finished session with no edits does not publish.
CLEAN_CAT="$ROOT/clean.yaml"
cat >"$CLEAN_CAT" <<'EOF'
models:
  - id: "clean/three:free"
EOF
CREPO="$(mktemp -d)"
init_repo "$CREPO"
COUT="$(mktemp)"
CGH="$(mktemp)"
(
	cd "$CREPO"
	PATH="$ROOT:$PATH" \
		GH_ARGS="$CGH" \
		OPENROUTER_API_KEY=test \
		BUG_HUNT_CATALOG="$CLEAN_CAT" \
		BUG_HUNT_KDEPS="$ROOT/fake-kdeps" \
		BUG_HUNT_FAKE_LOG="$FAKE_LOG" \
		BUG_HUNT_RESET=1 \
		BUG_HUNT_SHUFFLE=0 \
		BUG_HUNT_TIMEOUT=30 \
		bash "$HUNT" run >"$COUT" 2>&1
)
if [ $? -eq 0 ] && grep -q "no product fix" "$COUT" && [ ! -f "$CREPO/hunted.txt" ] \
	&& grep -q "bug-hunt: issue https://example.test/issues/3" "$COUT" \
	&& grep -q "issue create" "$CGH" \
	&& ! grep -q "pr create" "$CGH"; then
	test_passed "bug-hunt run with no edits skips the pull request and files an issue"
else
	test_failed "bug-hunt run with no edits skips the pull request and files an issue" "$(cat "$COUT"; echo ---; cat "$CGH")"
fi

# A provider-only failure prints no issue line, and the job does not file one.
REF_CAT="$ROOT/refused.yaml"
cat >"$REF_CAT" <<'EOF'
models:
  - id: "refused/four:free"
EOF
RGH="$(mktemp)"
ROUT="$(mktemp)"
(
	cd "$REPO"
	PATH="$ROOT:$PATH" \
		GH_ARGS="$RGH" \
		OPENROUTER_API_KEY=test \
		BUG_HUNT_CATALOG="$REF_CAT" \
		BUG_HUNT_KDEPS="$ROOT/fake-kdeps" \
		BUG_HUNT_FAKE_LOG="$FAKE_LOG" \
		BUG_HUNT_RESET=1 \
		BUG_HUNT_SHUFFLE=0 \
		BUG_HUNT_MAX_MODELS=1 \
		BUG_HUNT_TASKS=1 \
		BUG_HUNT_TIMEOUT=30 \
		bash "$HUNT" run >"$ROUT" 2>&1
) || true
if grep -q "auto-improve: no issue" "$ROUT" && ! grep -q "issue create" "$RGH"; then
	test_passed "bug-hunt does not file an issue for provider-only failures"
else
	test_failed "bug-hunt does not file an issue for provider-only failures" "$(cat "$ROUT"; echo ---; cat "$RGH")"
fi

# Every id fails.
FAIL_CAT="$ROOT/fail.yaml"
cat >"$FAIL_CAT" <<'EOF'
models:
  - id: "fail/one:free"
EOF
FREPO="$(mktemp -d)"
init_repo "$FREPO"
FOUT="$(mktemp)"
FGH="$(mktemp)"
(
	cd "$FREPO"
	PATH="$ROOT:$PATH" \
		GH_ARGS="$FGH" \
		OPENROUTER_API_KEY=test \
		BUG_HUNT_CATALOG="$FAIL_CAT" \
		BUG_HUNT_KDEPS="$ROOT/fake-kdeps" \
		BUG_HUNT_FAKE_LOG="$FAKE_LOG" \
		BUG_HUNT_RESET=1 \
		BUG_HUNT_SHUFFLE=0 \
		BUG_HUNT_TIMEOUT=30 \
		bash "$HUNT" run >"$FOUT" 2>&1
)
if [ $? -ne 0 ] && grep -q "every tried free model failed" "$FOUT" \
	&& grep -q "bug-hunt: issue https://example.test/issues/3" "$FOUT" \
	&& grep -q "issue create" "$FGH"; then
	test_passed "bug-hunt run fails when every free model fails"
else
	test_failed "bug-hunt run fails when every free model fails" "$(cat "$FOUT"; echo ---; cat "$FGH")"
fi

# A failed gh issue create stops the hunt.
IREPO="$(mktemp -d)"
init_repo "$IREPO"
IOUT="$(mktemp)"
(
	cd "$IREPO"
	PATH="$ROOT:$PATH" \
		GH_ARGS="$FGH" \
		GH_ISSUE_FAIL=1 \
		OPENROUTER_API_KEY=test \
		BUG_HUNT_CATALOG="$CLEAN_CAT" \
		BUG_HUNT_KDEPS="$ROOT/fake-kdeps" \
		BUG_HUNT_FAKE_LOG="$FAKE_LOG" \
		BUG_HUNT_SHUFFLE=0 \
		BUG_HUNT_TIMEOUT=30 \
		bash "$HUNT" run >"$IOUT" 2>&1
)
if [ $? -ne 0 ] && grep -q "issue failed" "$IOUT"; then
	test_passed "bug-hunt run stops when filing the issue fails"
else
	test_failed "bug-hunt run stops when filing the issue fails" "$(cat "$IOUT")"
fi

# Publish commits, pushes, opens a PR, and turns on auto-merge.
BARE="$(mktemp -d)"
git init -q --bare "$BARE"
PREPO="$(mktemp -d)"
init_repo "$PREPO"
git -C "$PREPO" remote add origin "$BARE"
echo dirty >"$PREPO/hunted.txt"
GH_ARGS="$(mktemp)"
POUT="$(mktemp)"
(
	cd "$PREPO"
	PATH="$ROOT:$PATH" \
		GH_ARGS="$GH_ARGS" \
		bash "$HUNT" publish "edit/two:free" >"$POUT" 2>&1
)
PUB_RC=$?
BRANCH="$(git -C "$PREPO" rev-parse --abbrev-ref HEAD)"
if [ "$PUB_RC" -eq 0 ] \
	&& printf '%s' "$BRANCH" | grep -q '^bot/bug-hunt-' \
	&& git --git-dir="$BARE" show-ref --verify --quiet "refs/heads/$BRANCH" \
	&& grep -q "pr create" "$GH_ARGS" \
	&& grep -q "pr merge --auto --squash" "$GH_ARGS" \
	&& grep -q "opened https://example.test/pull/9" "$POUT"; then
	test_passed "bug-hunt publish pushes a pull request and enables auto-merge"
else
	test_failed "bug-hunt publish pushes a pull request and enables auto-merge" "$(cat "$POUT"; echo ---; cat "$GH_ARGS"; echo ---; echo "rc=$PUB_RC branch=$BRANCH")"
fi

# An already-open bug-hunt pull request does not block the next hunt's pull request.
OBARE="$(mktemp -d)"
git init -q --bare "$OBARE"
OREPO="$(mktemp -d)"
init_repo "$OREPO"
git -C "$OREPO" remote add origin "$OBARE"
echo dirty >"$OREPO/hunted.txt"
OH_ARGS="$(mktemp)"
OOUT="$(mktemp)"
(
	cd "$OREPO"
	PATH="$ROOT:$PATH" \
		GH_ARGS="$OH_ARGS" \
		GH_OPEN_PR=1 \
		bash "$HUNT" publish "edit/two:free" "2" >"$OOUT" 2>&1
)
ORC=$?
OBRANCH="$(git -C "$OREPO" rev-parse --abbrev-ref HEAD)"
if [ "$ORC" -eq 0 ] && printf '%s' "$OBRANCH" | grep -q '^bot/bug-hunt-.*-2$' \
	&& grep -q "pr create" "$OH_ARGS" && ! grep -q "already exists" "$OOUT"; then
	test_passed "bug-hunt publish opens a pull request while another is open"
else
	test_failed "bug-hunt publish opens a pull request while another is open" "$(cat "$OOUT"; echo ---; cat "$OH_ARGS"; echo ---; echo "$OBRANCH")"
fi

rm -rf "$REPO" "$CREPO" "$FREPO" "$IREPO" "$PREPO" "$OREPO" "$BARE" "$OBARE" "$ROOT"
rm -f "$FAKE_LOG" "$OUT" "$COUT" "$FOUT" "$IOUT" "$GH_ARGS" "$CGH" "$FGH" "$POUT" "$OH_ARGS" "$OOUT"

echo ""
