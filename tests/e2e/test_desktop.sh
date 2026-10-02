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

for f in index.html style.css markdown.js app.js settings.js composer.js repl.js; do
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
    for f in markdown.js settingctl.js composer.js app.js settings.js repl.js; do
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
    if node "$DESKTOP_DIR/frontend/test/composer_test.js" >/dev/null 2>&1; then
        test_passed "desktop - autocomplete and model picker tests pass"
    else
        test_failed "desktop - autocomplete and model picker tests failed"
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

# Dialogs must not be capped narrower than the settings width, or form rows overflow.
if grep -Eq '^#settings \{[^}]*max-width: 94vw' "$DESKTOP_DIR/frontend/dist/style.css" \
    && grep -Eq 'minmax\(0, 1fr\)' "$DESKTOP_DIR/frontend/dist/style.css"; then
    test_passed "desktop - settings dialog width and field grid cannot overflow"
else
    test_failed "desktop - settings dialog or field grid can overflow"
fi

# Release builds cover Apple Silicon and Intel Macs.
if grep -q 'goarch: amd64' "$DESKTOP_DIR/../.github/workflows/release-desktop.yml" \
    && grep -q 'goarch: arm64' "$DESKTOP_DIR/../.github/workflows/release-desktop.yml"; then
    test_passed "desktop - release workflow builds macOS arm64 and amd64"
else
    test_failed "desktop - release workflow is missing a macOS architecture"
fi

# Harness cards lay their toggles out in one aligned group, with checkboxes reset to a fixed size.
if grep -q 'hs-toggles' "$DESKTOP_DIR/frontend/dist/settings.js" \
    && grep -q '^\.hs-toggles' "$DESKTOP_DIR/frontend/dist/style.css" \
    && grep -q '^input\[type="checkbox"\]' "$DESKTOP_DIR/frontend/dist/style.css"; then
    test_passed "desktop - harness toggles are grouped and aligned"
else
    test_failed "desktop - harness toggle layout missing"
fi

# App icon on every platform: Windows exe resource, Linux window icon + launcher, in-app logo.
if [ -s "$DESKTOP_DIR/rsrc_windows_amd64.syso" ] \
    && grep -q 'go:embed packaging/icon.png' "$DESKTOP_DIR/main.go" \
    && [ -s "$DESKTOP_DIR/packaging/linux/kdeps.png" ] \
    && grep -q '^Icon=kdeps' "$DESKTOP_DIR/packaging/linux/kdeps-desktop.desktop" \
    && [ -s "$DESKTOP_DIR/frontend/dist/icon.png" ] \
    && grep -q 'rel="icon"' "$DESKTOP_DIR/frontend/dist/index.html"; then
    test_passed "desktop - app icon wired for macOS, Windows, Linux and the window"
else
    test_failed "desktop - app icon missing on some platform"
fi

# Homebrew cask: generator emits a valid cask for both architectures, and the release workflow publishes it.
CASK_OUT="$("$DESKTOP_DIR/packaging/macos/make-cask.sh" v1.2.3 aaa111 bbb222 2>/dev/null)"
if echo "$CASK_OUT" | grep -q '^cask "kdeps-desktop" do' \
    && echo "$CASK_OUT" | grep -q 'version "1.2.3"' \
    && echo "$CASK_OUT" | grep -q 'sha256 "aaa111"' \
    && echo "$CASK_OUT" | grep -q 'sha256 "bbb222"' \
    && echo "$CASK_OUT" | grep -q 'darwin_arm64.dmg' && echo "$CASK_OUT" | grep -q 'darwin_amd64.dmg' \
    && echo "$CASK_OUT" | grep -q 'app "kdeps.app"' \
    && grep -q 'make-cask.sh' "$DESKTOP_DIR/../.github/workflows/release-desktop.yml"; then
    test_passed "desktop - homebrew cask generated and published by the release workflow"
else
    test_failed "desktop - homebrew cask generation or publishing missing"
fi

# Wide distribution: Scoop manifest, AUR package files, native Linux packages, goreleaser publishers.
PKG="$DESKTOP_DIR/packaging"
SCOOP_OUT="$("$PKG/windows/make-scoop.sh" v1.2.3 abc 2>/dev/null)"
AUR_DIR="$(mktemp -d)"
"$PKG/linux/make-aur.sh" v1.2.3 abc "$AUR_DIR" >/dev/null 2>&1
if echo "$SCOOP_OUT" | grep -q 'windows_amd64.zip' && echo "$SCOOP_OUT" | grep -q '"hash": "abc"' \
    && grep -q '^pkgname=kdeps-desktop-bin' "$AUR_DIR/PKGBUILD" && grep -q 'pkgver = 1.2.3' "$AUR_DIR/.SRCINFO" \
    && grep -q 'libwebkit2gtk-4.1-0' "$PKG/linux/nfpm.yaml" \
    && grep -q 'nfpm package' "$DESKTOP_DIR/../.github/workflows/release-desktop.yml" \
    && grep -q 'make-scoop.sh' "$DESKTOP_DIR/../.github/workflows/release-desktop.yml" \
    && grep -q 'make-aur.sh' "$DESKTOP_DIR/../.github/workflows/release-desktop.yml" \
    && grep -q '^scoops:' "$DESKTOP_DIR/../.goreleaser.yaml" && grep -q '^aurs:' "$DESKTOP_DIR/../.goreleaser.yaml" \
    && grep -q '^winget:' "$DESKTOP_DIR/../.goreleaser.yaml"; then
    test_passed "desktop - scoop, AUR, deb/rpm/arch and winget publishing wired"
else
    test_failed "desktop - package manager publishing is incomplete"
fi
rm -rf "$AUR_DIR"

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
