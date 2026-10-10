#!/usr/bin/env bash
# Fails when a versioned asset file changed between BASE and HEAD without a
# higher `version:` header. Template items are versioned by their
# template.yaml, so any change inside a template directory needs a bump there.
#
# usage: scripts/check-asset-versions.sh <base-ref> <head-ref>
set -euo pipefail

base=${1:?base ref}
head=${2:?head ref}

flat_re='^(pkg/agent/(harness|events|actions|presets|themes|tools)|pkg/llmserver/catalog/recipes)/[^/]+\.yaml$'
tmpl_re='^pkg/templates/templates/([^/]+)/'

version_at() { # <ref> <path> -> version or empty
	git show "$1:$2" 2>/dev/null | sed -n 's/^version:[[:space:]]*"\{0,1\}\([^"[:space:]]*\)"\{0,1\}[[:space:]]*$/\1/p' | head -n1
}

newer() { # <a> <b>: a is a strictly higher version than b
	[ "$1" != "$2" ] && [ "$(printf '%s\n%s\n' "$1" "$2" | sort -V | tail -n1)" = "$1" ]
}

check() { # <file holding the version> <label>
	local file=$1 label=$2 old new
	new=$(version_at "$head" "$file")
	if [ -z "$new" ]; then
		echo "::error file=$file::$label has no version header"
		return 1
	fi
	old=$(version_at "$base" "$file")
	if [ -n "$old" ] && ! newer "$new" "$old"; then
		echo "::error file=$file::$label changed but its version stayed $old (now $new); bump it"
		return 1
	fi
	return 0
}

failed=0
seen=" "
while IFS= read -r f; do
	if [[ $f =~ $flat_re ]]; then
		check "$f" "$f" || failed=1
	elif [[ $f =~ $tmpl_re ]]; then
		item=${BASH_REMATCH[1]}
		case $seen in *" $item "*) continue ;; esac
		seen="$seen$item "
		manifest="pkg/templates/templates/$item/template.yaml"
		git cat-file -e "$head:pkg/templates/templates/$item" 2>/dev/null || continue # template deleted
		check "$manifest" "template $item" || failed=1
	fi
done < <(git diff --name-only --diff-filter=AMR "$base" "$head")

if [ "$failed" -ne 0 ]; then
	exit 1
fi
echo "asset versions OK"
