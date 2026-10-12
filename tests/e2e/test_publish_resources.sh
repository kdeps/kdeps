# shellcheck shell=bash
# E2E: tools/publish-assets -resources publishes one docs page and one
# registry formula per built-in resource type, at the kdeps docs version.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing publish-assets resource types..."

if ! command -v go >/dev/null 2>&1; then
    test_skipped "publish resources - go not available"
    return 0 2>/dev/null || exit 0
fi

PR_DIR=$(mktemp -d)
OUT=$(cd "$PROJECT_ROOT" && go run ./tools/publish-assets -out "$PR_DIR/assets" -formulas "$PR_DIR/formulas" \
    -resources "$PR_DIR/resources" -docs docs/v2 -commit e2e 2>&1)
PR_VERSION=$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["version"])' "$PROJECT_ROOT/docs/v2/package.json" 2>/dev/null)
PR_FORMULA="$PR_DIR/formulas/kdeps-resource-scraper.yaml"

if [ -f "$PR_DIR/resources/scraper/$PR_VERSION.md" ] && grep -q "^type: resource$" "$PR_FORMULA" 2>/dev/null \
    && grep -q "resources/scraper/$PR_VERSION.md" "$PR_FORMULA"; then
    test_passed "publish-assets -resources writes the scraper docs page and formula"
else
    test_failed "publish-assets -resources" "$OUT"
fi

if [ "$(find "$PR_DIR/formulas" -name 'kdeps-resource-*.yaml' | wc -l | tr -d ' ')" -ge 20 ]; then
    test_passed "publish-assets -resources writes a formula per resource type"
else
    test_failed "publish-assets -resources formula count" "$(find "$PR_DIR/formulas" -name 'kdeps-resource-*')"
fi

rm -rf "$PR_DIR"
