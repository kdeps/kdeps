#!/usr/bin/env bash
# Daily bug hunt: run /auto-improve through kdeps on an OpenRouter free model.
# A session that does not finish tries the next id from the free catalog.
# BUG_HUNT_FOCUS limits the session to one id from the focus list. Each focus
# is its own branch and its own pull request. List order is priority.
# A tree with edits is pushed as a pull request with auto-merge.
#
#   scripts/bug-hunt.sh models
#   scripts/bug-hunt.sh focuses
#   scripts/bug-hunt.sh focuses-json
#   scripts/bug-hunt.sh run
#   scripts/bug-hunt.sh publish <model-id>
#
# OPENROUTER_API_KEY is required for run.
# BUG_HUNT_CATALOG overrides assets/models/openrouter.yaml.
# BUG_HUNT_FOCUSES overrides assets/commands/auto-improve.yaml.
# BUG_HUNT_FOCUS is one focus id. Unset runs /auto-improve with no focus.
# BUG_HUNT_KDEPS overrides ./kdeps.
# BUG_HUNT_MAX_MODELS caps how many ids are tried (default 4).
# BUG_HUNT_RESET=1 discards a failed model's edits before the next id.
# BUG_HUNT_SKIP_PUBLISH=1 leaves edits in the tree and does not call gh.
# GH_TOKEN must be able to push the branch and open the pull request.
# A GITHUB_TOKEN push does not start other workflows, so CI never goes green
# and auto-merge never lands. Use a token that does start them (RELEASE_TOKEN).

set -euo pipefail

catalog="${BUG_HUNT_CATALOG:-assets/models/openrouter.yaml}"
focus_file="${BUG_HUNT_FOCUSES:-assets/commands/auto-improve.yaml}"
kdeps_bin="${BUG_HUNT_KDEPS:-./kdeps}"
max_models="${BUG_HUNT_MAX_MODELS:-4}"

models() {
	grep -E '^[[:space:]]*- id: "' "$catalog" | sed -E 's/.*id: "([^"]+)".*/\1/'
}

# focuses prints focus ids in file order. That order is priority.
focuses() {
	awk '
		/^focus:/ { in_focus = 1; next }
		in_focus && /^[^ ]/ { in_focus = 0 }
		in_focus && /^  - id: / {
			id = $0
			sub(/^  - id: /, "", id)
			gsub(/"/, "", id)
			print id
		}
	' "$focus_file"
}

focuses_json() {
	local first=1 id
	printf '['
	while IFS= read -r id; do
		[ -n "$id" ] || continue
		if [ "$first" -eq 1 ]; then
			first=0
		else
			printf ','
		fi
		printf '"%s"' "$id"
	done < <(focuses)
	printf ']\n'
}

require_focus() {
	[ -z "${BUG_HUNT_FOCUS:-}" ] && return 0
	if ! focuses | grep -qx "${BUG_HUNT_FOCUS}"; then
		echo "bug-hunt: unknown focus ${BUG_HUNT_FOCUS}" >&2
		return 1
	fi
}

tree_dirty() {
	[ -n "$(git status --porcelain)" ]
}

# discard_attempt drops edits from a model that did not finish.
discard_attempt() {
	if [ "${BUG_HUNT_RESET:-}" != "1" ]; then
		echo "bug-hunt: leaving the tree as the failed model left it (set BUG_HUNT_RESET=1 to discard)"
		return 0
	fi
	git reset --hard
	git clean -fd
}

publish() {
	local model="$1"
	require_focus || return 1
	if [ -n "${BUG_HUNT_SKIP_PUBLISH:-}" ]; then
		echo "bug-hunt: skip publish"
		return 0
	fi
	if ! tree_dirty; then
		echo "bug-hunt: no product fix"
		return 0
	fi
	if ! command -v gh >/dev/null 2>&1; then
		echo "bug-hunt: gh is required to open the pull request" >&2
		return 1
	fi
	local needle="bot/bug-hunt"
	if [ -n "${BUG_HUNT_FOCUS:-}" ]; then
		needle="bot/bug-hunt-${BUG_HUNT_FOCUS}-"
	fi
	if gh pr list --state open --json headRefName | grep -q "$needle"; then
		echo "bug-hunt: an open bug-hunt pull request already exists"
		return 0
	fi
	local day branch label
	day="$(date -u +%Y%m%d%H%M%S)"
	label="bug hunt"
	branch="bot/bug-hunt-${day}"
	if [ -n "${BUG_HUNT_FOCUS:-}" ]; then
		label="bug hunt ${BUG_HUNT_FOCUS}"
		branch="bot/bug-hunt-${BUG_HUNT_FOCUS}-${day}"
	fi
	git checkout -b "$branch" || return 1
	git add -A -- . ':(exclude)private' ':(exclude).env' ':(exclude)*.env'
	if git diff --cached --quiet; then
		echo "bug-hunt: no product fix"
		return 0
	fi
	git -c user.name='github-actions[bot]' \
		-c user.email='github-actions[bot]@users.noreply.github.com' \
		commit -m "fix: ${label} ${day}" \
		-m "Automated /auto-improve session on OpenRouter model ${model}." || return 1
	git push -u origin "$branch" || return 1
	local body url
	body="$(mktemp)"
	cat >"$body" <<EOF
Daily bug hunt. kdeps ran /auto-improve${BUG_HUNT_FOCUS:+ ${BUG_HUNT_FOCUS}} with OpenRouter model \`${model}\` and edited the tree.

Auto-merge is on. The pull request merges after the required checks pass.
EOF
	url="$(gh pr create --base main --head "$branch" --title "fix: ${label} ${day}" --body-file "$body")" || {
		rm -f "$body"
		return 1
	}
	rm -f "$body"
	echo "bug-hunt: opened ${url}"
	# Enabling auto-merge fails while the PR is still "unstable". The required
	# checks then merge it once they pass. A transient failure stays in the log.
	gh pr merge --auto --squash "$url" || echo "bug-hunt: auto-merge not enabled yet for ${url}"
}

run_one() {
	local model="$1"
	local log
	log="$(mktemp)"
	local session="bug-hunt" slash="/auto-improve"
	if [ -n "${BUG_HUNT_FOCUS:-}" ]; then
		session="bug-hunt-${BUG_HUNT_FOCUS}"
		slash="/auto-improve ${BUG_HUNT_FOCUS}"
	fi
	set +e
	if command -v timeout >/dev/null 2>&1; then
		printf '%s\n/exit\n' "$slash" | timeout "${BUG_HUNT_TIMEOUT:-20m}" \
			"$kdeps_bin" --backend openrouter --model "$model" --instrument="$session" \
			>"$log" 2>&1
	else
		printf '%s\n/exit\n' "$slash" | \
			"$kdeps_bin" --backend openrouter --model "$model" --instrument="$session" \
			>"$log" 2>&1
	fi
	local code=$?
	cat "$log"
	if grep -q 'auto-improve: done' "$log"; then
		rm -f "$log"
		if tree_dirty; then
			publish "$model" || return 2
		else
			echo "bug-hunt: no product fix"
		fi
		return 0
	fi
	rm -f "$log"
	echo "bug-hunt: ${model} did not finish (exit ${code}), trying the next free model"
	discard_attempt
	return 1
}

run_hunt() {
	if [ -z "${OPENROUTER_API_KEY:-}" ]; then
		echo "missing the openrouter API key, set it in the OPENROUTER_API_KEY environment variable" >&2
		return 1
	fi
	require_focus || return 1
	if [ ! -f "$catalog" ]; then
		echo "bug-hunt: no OpenRouter free models in ${catalog}" >&2
		return 1
	fi
	local id n=0 tried=0 rc
	while IFS= read -r id; do
		[ -n "$id" ] || continue
		n=$((n + 1))
		if [ "$n" -gt "$max_models" ]; then
			break
		fi
		tried=1
		echo "bug-hunt: trying ${id}"
		set +e
		run_one "$id"
		rc=$?
		set -e
		if [ "$rc" -eq 0 ]; then
			return 0
		fi
		if [ "$rc" -eq 2 ]; then
			return 1
		fi
	done < <(models)
	if [ "$tried" -eq 0 ]; then
		echo "bug-hunt: no OpenRouter free models in ${catalog}" >&2
		return 1
	fi
	echo "bug-hunt: every tried free model failed" >&2
	return 1
}

case "${1:-}" in
models) models ;;
focuses) focuses ;;
focuses-json) focuses_json ;;
run) run_hunt ;;
publish)
	if [ -z "${2:-}" ]; then
		echo "usage: scripts/bug-hunt.sh publish <model-id>" >&2
		exit 1
	fi
	publish "$2"
	;;
*)
	echo "usage: scripts/bug-hunt.sh models|focuses|focuses-json|run|publish <model-id>" >&2
	exit 1
	;;
esac
