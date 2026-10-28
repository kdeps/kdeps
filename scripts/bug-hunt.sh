#!/usr/bin/env bash
# Daily bug hunt: one task per process, then evaluate, fix, and a pull request.
# run shuffles the free catalog. models prints catalog order.
# BUG_HUNT_SHUFFLE=0 keeps catalog order for run.
# BUG_HUNT_TASKS is how many hunts to run (default 3, the /auto-improve list).
# Each hunt is a new process and a new instrument session (bug-hunt-N), so the
# log holds that task and its evaluate only. After a hunt finishes, a dirty
# tree is pushed as its own pull request, then the next hunt starts on the
# branch the job checked out.
# Inside a hunt, the task and the evaluate turn jump to another free model
# at random. A provider 429 during the capture does the same.
# Free-model requests wait so they stay under the account per-minute cap.
# An account 429 waits out the minute and retries the same id.
# The command files one GitHub issue per hunt. If the process dies before
# printing auto-improve: issue <url>, this script files that issue.
#
#   scripts/bug-hunt.sh models
#   scripts/bug-hunt.sh run
#   scripts/bug-hunt.sh publish <model-id>
#
# OPENROUTER_API_KEY is required for run.
# BUG_HUNT_CATALOG overrides assets/models/openrouter.yaml.
# BUG_HUNT_KDEPS overrides ./kdeps.
# BUG_HUNT_MAX_MODELS caps how many ids are tried per hunt (default 4).
# BUG_HUNT_TASKS is the number of hunts (default 3).
# BUG_HUNT_RESET=1 discards a failed model's edits before the next id.
# BUG_HUNT_SKIP_PUBLISH=1 leaves edits in the tree and does not open a pull request.
# GH_TOKEN must be able to push the branch, open the pull request, and file issues.
# A GITHUB_TOKEN push does not start other workflows, so CI never goes green
# and auto-merge never lands. Use a token that does start them (RELEASE_TOKEN).

set -euo pipefail

catalog="${BUG_HUNT_CATALOG:-assets/models/openrouter.yaml}"
kdeps_bin="${BUG_HUNT_KDEPS:-./kdeps}"
max_models="${BUG_HUNT_MAX_MODELS:-4}"
tasks="${BUG_HUNT_TASKS:-3}"

models() {
	grep -E '^[[:space:]]*- id: "' "$catalog" | sed -E 's/.*id: "([^"]+)".*/\1/'
}

# shuffled_models is the try order for run. A new awk process seeds srand.
shuffled_models() {
	if [ "${BUG_HUNT_SHUFFLE:-1}" = "0" ]; then
		models
		return 0
	fi
	models | awk 'BEGIN { srand() } { printf "%s\t%s\n", rand(), $0 }' | sort -n | cut -f2-
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
	local task="${2:-}"
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
	local day branch
	day="$(date -u +%Y%m%d%H%M%S)"
	branch="bot/bug-hunt-${day}"
	if [ -n "$task" ]; then
		branch="${branch}-${task}"
	fi
	git checkout -b "$branch" || return 1
	git add -A -- . ':(exclude)private' ':(exclude).env' ':(exclude)*.env'
	if git diff --cached --quiet; then
		echo "bug-hunt: no product fix"
		return 0
	fi
	git -c user.name='github-actions[bot]' \
		-c user.email='github-actions[bot]@users.noreply.github.com' \
		commit -m "fix: bug hunt ${day}" \
		-m "Automated /auto-improve session on OpenRouter model ${model}." || return 1
	git push -u origin "$branch" || return 1
	local body url
	body="$(mktemp)"
	cat >"$body" <<EOF
Daily bug hunt. kdeps ran /auto-improve with OpenRouter model \`${model}\` and edited the tree.

Auto-merge is on. The pull request merges after the required checks pass.
EOF
	url="$(gh pr create --base main --head "$branch" --title "fix: bug hunt ${day}" --body-file "$body")" || {
		rm -f "$body"
		return 1
	}
	rm -f "$body"
	echo "bug-hunt: opened ${url}"
	# Enabling auto-merge fails while the PR is still "unstable". The required
	# checks then merge it once they pass. A transient failure stays in the log.
	gh pr merge --auto --squash "$url" || echo "bug-hunt: auto-merge not enabled yet for ${url}"
}

# file_issue opens one issue when the process did not print an issue URL.
# A printed auto-improve: issue http line means the command already filed it.
file_issue() {
	local model="$1"
	local log="$2"
	if grep -q 'auto-improve: issue http' "$log"; then
		return 0
	fi
	if ! command -v gh >/dev/null 2>&1; then
		echo "bug-hunt: gh is required to file the issue" >&2
		return 1
	fi
	local title body errf url
	title="bug: capture $(date -u +%Y-%m-%dT%H:%MZ) ${model}"
	body="$(mktemp)"
	errf="$(mktemp)"
	{
		printf 'Daily bug hunt on OpenRouter model `%s` ended without an issue URL.\n\n' "$model"
		printf '```\n'
		tail -n 80 "$log"
		printf '```\n'
	} >"$body"
	url="$(gh issue create --title "$title" --body-file "$body" 2>"$errf")" || {
		echo "bug-hunt: issue failed: $(tr '\n' ' ' <"$errf")" >&2
		rm -f "$body" "$errf"
		return 1
	}
	rm -f "$body" "$errf"
	echo "bug-hunt: issue ${url}"
}

run_one() {
	local model="$1"
	local task="$2"
	local log
	log="$(mktemp)"
	set +e
	if command -v timeout >/dev/null 2>&1; then
		printf '/auto-improve %s\n/exit\n' "$task" | timeout "${BUG_HUNT_TIMEOUT:-20m}" \
			"$kdeps_bin" --backend openrouter --model "$model" --instrument="bug-hunt-${task}" \
			>"$log" 2>&1
	else
		printf '/auto-improve %s\n/exit\n' "$task" | \
			"$kdeps_bin" --backend openrouter --model "$model" --instrument="bug-hunt-${task}" \
			>"$log" 2>&1
	fi
	local code=$?
	cat "$log"
	if ! file_issue "$model" "$log"; then
		rm -f "$log"
		return 2
	fi
	if grep -q 'auto-improve: done' "$log"; then
		rm -f "$log"
		if tree_dirty; then
			publish "$model" "$task" || return 2
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
	if [ ! -f "$catalog" ]; then
		echo "bug-hunt: no OpenRouter free models in ${catalog}" >&2
		return 1
	fi
	local base=""
	if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
		base="$(git rev-parse --abbrev-ref HEAD)"
	fi
	local task finished=0 failed=0
	for task in $(seq 1 "$tasks"); do
		echo "bug-hunt: hunt ${task}/${tasks}"
		local id n=0 tried=0 rc done_task=0
		while IFS= read -r id; do
			[ -n "$id" ] || continue
			n=$((n + 1))
			if [ "$n" -gt "$max_models" ]; then
				break
			fi
			tried=1
			echo "bug-hunt: trying ${id}"
			set +e
			run_one "$id" "$task"
			rc=$?
			set -e
			if [ "$rc" -eq 0 ]; then
				done_task=1
				finished=1
				break
			fi
			if [ "$rc" -eq 2 ]; then
				return 1
			fi
		done < <(shuffled_models)
		if [ "$tried" -eq 0 ]; then
			echo "bug-hunt: no OpenRouter free models in ${catalog}" >&2
			return 1
		fi
		if [ "$done_task" -eq 0 ]; then
			failed=1
			echo "bug-hunt: hunt ${task}: every tried free model failed" >&2
		fi
		if [ -n "$base" ] && [ "$(git rev-parse --abbrev-ref HEAD)" != "$base" ]; then
			if [ -z "$(git status --porcelain)" ]; then
				git checkout "$base"
			else
				echo "bug-hunt: staying on $(git rev-parse --abbrev-ref HEAD); the tree still has edits"
			fi
		fi
	done
	if [ "$finished" -eq 0 ]; then
		echo "bug-hunt: every tried free model failed" >&2
		return 1
	fi
	if [ "$failed" -eq 1 ]; then
		return 1
	fi
}

case "${1:-}" in
models) models ;;
run) run_hunt ;;
publish)
	if [ -z "${2:-}" ]; then
		echo "usage: scripts/bug-hunt.sh publish <model-id>" >&2
		exit 1
	fi
	publish "$2" "${3:-}"
	;;
*)
	echo "usage: scripts/bug-hunt.sh models|run|publish <model-id>" >&2
	exit 1
	;;
esac
