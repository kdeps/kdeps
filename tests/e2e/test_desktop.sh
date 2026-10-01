# shellcheck shell=bash
# E2E: desktop GUI (desktop/ Wails module). A window cannot be driven headless,
# so this checks what a release depends on: the embedded front end is complete
# and parses, every script/style index.html references exists, and the shell
# compiles with the production tags where a native toolchain is available.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing desktop GUI..."

DESKTOP_DIR="$SCRIPT_DIR/../../desktop"
DIST="$DESKTOP_DIR/frontend/dist"

for f in index.html style.css markdown.js app.js settings.js; do
    if [ -s "$DIST/$f" ]; then
        test_passed "desktop - embedded asset $f present"
    else
        test_failed "desktop - embedded asset $f missing or empty" "$DIST/$f"
    fi
done

MISSING=""
for ref in $(grep -oE '(src|href)="[^"]+\.(js|css)"' "$DIST/index.html" | sed -E 's/^(src|href)="//; s/"$//'); do
    [ -f "$DIST/$ref" ] || MISSING="$MISSING $ref"
done
if [ -z "$MISSING" ]; then
    test_passed "desktop - index.html references only existing assets"
else
    test_failed "desktop - index.html references missing assets" "$MISSING"
fi

if grep -Eq 'https?://[^"]*\.(js|css)"' "$DIST/index.html"; then
    test_failed "desktop - index.html loads a remote script or style (must work offline)"
else
    test_passed "desktop - front end is fully offline (no CDN references)"
fi

if command -v node >/dev/null 2>&1; then
    SYNTAX_OK=1
    for f in markdown.js settingctl.js app.js settings.js; do
        node --check "$DIST/$f" >/dev/null 2>&1 || SYNTAX_OK=0
    done
    if [ "$SYNTAX_OK" = 1 ]; then
        test_passed "desktop - front end scripts parse"
    else
        test_failed "desktop - front end script has a syntax error"
    fi
    if node "$DESKTOP_DIR/frontend/test/markdown_test.js" >/dev/null 2>&1; then
        test_passed "desktop - markdown renderer tests (including XSS cases) pass"
    else
        test_failed "desktop - markdown renderer tests failed"
    fi
    if node "$DESKTOP_DIR/frontend/test/settingctl_test.js" >/dev/null 2>&1; then
        test_passed "desktop - settings control picker tests pass"
    else
        test_failed "desktop - settings control picker tests failed"
    fi
else
    test_skipped "desktop - node not installed, skipping script syntax check and markdown tests"
fi

CAN_BUILD=0
case "$(uname -s)" in
    Darwin) CAN_BUILD=1 ;;
    MINGW*|MSYS*|CYGWIN*) CAN_BUILD=1 ;;
    Linux) pkg-config --exists gtk+-3.0 webkit2gtk-4.1 2>/dev/null && CAN_BUILD=1 ;;
esac

if [ "$CAN_BUILD" = 1 ] && command -v go >/dev/null 2>&1; then
    DESKTOP_BIN="$(mktemp -d)/kdeps-desktop"
    TAGS="desktop,production"
    [ "$(uname -s)" = Linux ] && TAGS="$TAGS,webkit2_41"
    if [ "$(uname -s)" = Darwin ]; then
        export CGO_LDFLAGS="-framework UniformTypeIdentifiers"
    fi
    if (cd "$DESKTOP_DIR" && go build -tags "$TAGS" -o "$DESKTOP_BIN" . ) >/dev/null 2>&1; then
        test_passed "desktop - shell compiles with production tags"
    else
        test_failed "desktop - shell failed to compile"
    fi
    rm -rf "$(dirname "$DESKTOP_BIN")"
else
    test_skipped "desktop - no native webview toolchain, skipping shell compile"
fi

# Packaging files the Makefile and release workflow depend on.
[ -x "$DESKTOP_DIR/packaging/macos/package.sh" ] && [ -f "$DESKTOP_DIR/packaging/macos/Info.plist.in" ] \
    && [ -f "$DESKTOP_DIR/packaging/icon.png" ] \
    && test_passed "desktop - packaging files present" \
    || test_failed "desktop - packaging files missing or package.sh not executable"

if [ "$(uname -s)" = Darwin ] && [ "$CAN_BUILD" = 1 ] && command -v make >/dev/null 2>&1; then
    OUT_DIR="$(mktemp -d)"
    if make -C "$DESKTOP_DIR/.." desktop-package VERSION=0.0.0-e2e DESKTOP_OUT="$OUT_DIR" >/dev/null 2>&1 \
        && ls "$OUT_DIR"/kdeps-desktop_0.0.0-e2e_darwin_*.dmg >/dev/null 2>&1 \
        && [ -f "$OUT_DIR/kdeps.app/Contents/Info.plist" ] \
        && [ -f "$OUT_DIR/kdeps.app/Contents/Resources/icon.icns" ]; then
        test_passed "desktop - make desktop-package produces a dmg and app bundle"
    else
        test_failed "desktop - make desktop-package failed"
    fi
    rm -rf "$OUT_DIR"
else
    test_skipped "desktop - dmg packaging only runs on macOS"
fi
