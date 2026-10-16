#!/usr/bin/env bash
# Fails when a versioned asset file changed between BASE and HEAD without a
# higher `version:` header. Bundle items (templates, docker, k8s, schemas,
# components) are
# versioned by their template.yaml, so any change inside a bundle directory
# needs a bump there.
#
# usage: scripts/check-asset-versions.sh <base-ref> <head-ref>
set -euo pipefail

base=${1:?base ref}
head=${2:?head ref}

flat_re='^assets/(harness|events|actions|presets|themes|tools|commands|recipes|models|defaults|providers)/[^/]+\.yaml$'
bundle_re='^assets/(templates|docker|k8s|schemas|components)/([^/]+)/'

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
	elif [[ $f =~ $bundle_re ]]; then
		item="${BASH_REMATCH[1]}/${BASH_REMATCH[2]}"
		case $seen in *" $item "*) continue ;; esac
		seen="$seen$item "
		git cat-file -e "$head:assets/$item" 2>/dev/null || continue # bundle deleted
		check "assets/$item/template.yaml" "bundle $item" || failed=1
	fi
done < <(git diff --name-only --diff-filter=AMR "$base" "$head")

if [ "$failed" -ne 0 ]; then
	exit 1
fi
echo "asset versions OK"
