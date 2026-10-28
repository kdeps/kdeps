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

# The daily bug-hunt script: free-model ids, fallback to the next id,
# and a pull request with auto-merge when the tree changed.

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

cat >"$ROOT/focus.yaml" <<'EOF'
focus:
  - id: token-efficiency
    tasks:
      - look
    brief: tokens
  - id: ux-dx
    tasks:
      - look
    brief: ux
  - id: security
    tasks:
      - look
    brief: security
EOF

FOCUSES="$(BUG_HUNT_FOCUSES="$ROOT/focus.yaml" bash "$HUNT" focuses)"
FOCUS_JSON="$(BUG_HUNT_FOCUSES="$ROOT/focus.yaml" bash "$HUNT" focuses-json)"
if [ "$FOCUSES" = "$(printf '%s\n' token-efficiency ux-dx security)" ] \
	&& [ "$FOCUS_JSON" = '["token-efficiency","ux-dx","security"]' ]; then
	test_passed "bug-hunt focuses lists ids in priority order"
else
	test_failed "bug-hunt focuses lists ids in priority order" "$FOCUSES $FOCUS_JSON"
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
printf 'args:%s\n' "$*" >>"$BUG_HUNT_FAKE_LOG"
cat >>"$BUG_HUNT_FAKE_LOG"
case "$model" in
fail/*) exit 1 ;;
edit/*)
	echo "auto-improve: done"
	echo fixed >>hunted.txt
	exit 0
	;;
clean/*)
	echo "auto-improve: done"
	exit 0
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
		name="$GH_OPEN_PR"
		if [ "$name" = "1" ]; then
			name="bot/bug-hunt-existing"
		fi
		printf '%s\n' "[{\"headRefName\":\"${name}\"}]"
	else
		printf '%s\n' '[]'
	fi
	exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "create" ]; then
	echo "https://example.test/pull/9"
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

# An unknown focus exits before any model call.
UOUT="$(mktemp)"
UFAKE="$(mktemp)"
if OPENROUTER_API_KEY=test \
	BUG_HUNT_FOCUSES="$ROOT/focus.yaml" \
	BUG_HUNT_FOCUS=nope \
	BUG_HUNT_KDEPS="$ROOT/fake-kdeps" \
	BUG_HUNT_FAKE_LOG="$UFAKE" \
	bash "$HUNT" run >"$UOUT" 2>&1; then
	test_failed "bug-hunt run rejects an unknown focus" "exit 0"
else
	if grep -q "unknown focus nope" "$UOUT" && [ ! -s "$UFAKE" ]; then
		test_passed "bug-hunt run rejects an unknown focus"
	else
		test_failed "bug-hunt run rejects an unknown focus" "$(cat "$UOUT")"
	fi
fi
rm -f "$UOUT" "$UFAKE"

# First id fails, second finishes and edits. Publish is skipped.
REPO="$(mktemp -d)"
init_repo "$REPO"
FAKE_LOG="$(mktemp)"
OUT="$(mktemp)"
(
	cd "$REPO"
	OPENROUTER_API_KEY=test \
		BUG_HUNT_CATALOG="$ROOT/catalog.yaml" \
		BUG_HUNT_KDEPS="$ROOT/fake-kdeps" \
		BUG_HUNT_FAKE_LOG="$FAKE_LOG" \
		BUG_HUNT_RESET=1 \
		BUG_HUNT_SKIP_PUBLISH=1 \
		BUG_HUNT_MAX_MODELS=4 \
		BUG_HUNT_TIMEOUT=30 \
		bash "$HUNT" run >"$OUT" 2>&1
)
if [ $? -eq 0 ] && grep -q "fail/one:free" "$FAKE_LOG" && grep -q "edit/two:free" "$FAKE_LOG" \
	&& [ "$(cat "$REPO/hunted.txt" 2>/dev/null)" = "fixed" ] \
	&& grep -q "trying the next free model" "$OUT" \
	&& grep -q "skip publish" "$OUT"; then
	test_passed "bug-hunt run tries the next free model and keeps the fix"
else
	test_failed "bug-hunt run tries the next free model and keeps the fix" "$(cat "$OUT"; echo ---; cat "$FAKE_LOG")"
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
(
	cd "$CREPO"
	OPENROUTER_API_KEY=test \
		BUG_HUNT_CATALOG="$CLEAN_CAT" \
		BUG_HUNT_KDEPS="$ROOT/fake-kdeps" \
		BUG_HUNT_FAKE_LOG="$FAKE_LOG" \
		BUG_HUNT_RESET=1 \
		BUG_HUNT_TIMEOUT=30 \
		bash "$HUNT" run >"$COUT" 2>&1
)
if [ $? -eq 0 ] && grep -q "no product fix" "$COUT" && [ ! -f "$CREPO/hunted.txt" ]; then
	test_passed "bug-hunt run with no edits skips the pull request"
else
	test_failed "bug-hunt run with no edits skips the pull request" "$(cat "$COUT")"
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
(
	cd "$FREPO"
	OPENROUTER_API_KEY=test \
		BUG_HUNT_CATALOG="$FAIL_CAT" \
		BUG_HUNT_KDEPS="$ROOT/fake-kdeps" \
		BUG_HUNT_FAKE_LOG="$FAKE_LOG" \
		BUG_HUNT_RESET=1 \
		BUG_HUNT_TIMEOUT=30 \
		bash "$HUNT" run >"$FOUT" 2>&1
)
if [ $? -ne 0 ] && grep -q "every tried free model failed" "$FOUT"; then
	test_passed "bug-hunt run fails when every free model fails"
else
	test_failed "bug-hunt run fails when every free model fails" "$(cat "$FOUT")"
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

# A focused run passes that id to kdeps and keeps the edit.
EDIT_CAT="$ROOT/edit.yaml"
cat >"$EDIT_CAT" <<'EOF'
models:
  - id: "edit/two:free"
EOF
HREPO="$(mktemp -d)"
init_repo "$HREPO"
FFAKE="$(mktemp)"
FOUT2="$(mktemp)"
(
	cd "$HREPO"
	OPENROUTER_API_KEY=test \
		BUG_HUNT_CATALOG="$EDIT_CAT" \
		BUG_HUNT_FOCUSES="$ROOT/focus.yaml" \
		BUG_HUNT_FOCUS=token-efficiency \
		BUG_HUNT_KDEPS="$ROOT/fake-kdeps" \
		BUG_HUNT_FAKE_LOG="$FFAKE" \
		BUG_HUNT_RESET=1 \
		BUG_HUNT_SKIP_PUBLISH=1 \
		BUG_HUNT_TIMEOUT=30 \
		bash "$HUNT" run >"$FOUT2" 2>&1
)
if [ $? -eq 0 ] \
	&& grep -q "args:.*--instrument=bug-hunt-token-efficiency" "$FFAKE" \
	&& grep -q "/auto-improve token-efficiency" "$FFAKE" \
	&& [ "$(cat "$HREPO/hunted.txt" 2>/dev/null)" = "fixed" ]; then
	test_passed "bug-hunt run passes the focus id to kdeps"
else
	test_failed "bug-hunt run passes the focus id to kdeps" "$(cat "$FOUT2"; echo ---; cat "$FFAKE")"
fi

# An open bug-hunt pull request is left alone.
OREPO="$(mktemp -d)"
init_repo "$OREPO"
echo dirty >"$OREPO/hunted.txt"
OH_ARGS="$(mktemp)"
OOUT="$(mktemp)"
(
	cd "$OREPO"
	PATH="$ROOT:$PATH" \
		GH_ARGS="$OH_ARGS" \
		GH_OPEN_PR=1 \
		bash "$HUNT" publish "edit/two:free" >"$OOUT" 2>&1
)
if [ $? -eq 0 ] && grep -q "already exists" "$OOUT" && ! grep -q "pr create" "$OH_ARGS"; then
	test_passed "bug-hunt publish skips when a bug-hunt pull request is open"
else
	test_failed "bug-hunt publish skips when a bug-hunt pull request is open" "$(cat "$OOUT"; echo ---; cat "$OH_ARGS")"
fi

# Another focus's open pull request does not block this focus.
XBARE="$(mktemp -d)"
git init -q --bare "$XBARE"
XREPO="$(mktemp -d)"
init_repo "$XREPO"
git -C "$XREPO" remote add origin "$XBARE"
echo dirty >"$XREPO/hunted.txt"
XH_ARGS="$(mktemp)"
XOUT="$(mktemp)"
(
	cd "$XREPO"
	PATH="$ROOT:$PATH" \
		GH_ARGS="$XH_ARGS" \
		GH_OPEN_PR="bot/bug-hunt-security-existing" \
		BUG_HUNT_FOCUSES="$ROOT/focus.yaml" \
		BUG_HUNT_FOCUS=ux-dx \
		bash "$HUNT" publish "edit/two:free" >"$XOUT" 2>&1
)
XRC=$?
XBRANCH="$(git -C "$XREPO" rev-parse --abbrev-ref HEAD)"
if [ "$XRC" -eq 0 ] \
	&& printf '%s' "$XBRANCH" | grep -q '^bot/bug-hunt-ux-dx-' \
	&& grep -q "pr create" "$XH_ARGS"; then
	test_passed "bug-hunt publish keeps each focus independent"
else
	test_failed "bug-hunt publish keeps each focus independent" "$(cat "$XOUT"; echo ---; echo "rc=$XRC branch=$XBRANCH")"
fi

# The same focus's open pull request is left alone.
SREPO="$(mktemp -d)"
init_repo "$SREPO"
echo dirty >"$SREPO/hunted.txt"
SH_ARGS="$(mktemp)"
SOUT="$(mktemp)"
(
	cd "$SREPO"
	PATH="$ROOT:$PATH" \
		GH_ARGS="$SH_ARGS" \
		GH_OPEN_PR="bot/bug-hunt-security-existing" \
		BUG_HUNT_FOCUSES="$ROOT/focus.yaml" \
		BUG_HUNT_FOCUS=security \
		bash "$HUNT" publish "edit/two:free" >"$SOUT" 2>&1
)
if [ $? -eq 0 ] && grep -q "already exists" "$SOUT" && ! grep -q "pr create" "$SH_ARGS"; then
	test_passed "bug-hunt publish skips when that focus already has a pull request"
else
	test_failed "bug-hunt publish skips when that focus already has a pull request" "$(cat "$SOUT"; echo ---; cat "$SH_ARGS")"
fi

rm -rf "$REPO" "$CREPO" "$FREPO" "$PREPO" "$OREPO" "$BARE" "$HREPO" "$XBARE" "$XREPO" "$SREPO" "$ROOT"
rm -f "$FAKE_LOG" "$OUT" "$COUT" "$FOUT" "$GH_ARGS" "$POUT" "$OH_ARGS" "$OOUT" "$FFAKE" "$FOUT2" "$XH_ARGS" "$XOUT" "$SH_ARGS" "$SOUT"

echo ""
