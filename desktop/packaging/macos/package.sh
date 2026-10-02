#!/usr/bin/env bash
# Wrap a built kdeps-desktop binary into kdeps.app and a .dmg.
# Usage: package.sh <binary> <version> <out-dir>
# Signing: ad-hoc by default. Set APPLE_SIGN_IDENTITY to a "Developer ID
# Application: ..." identity for a real signature (notarization is separate).
set -euo pipefail

BIN="$1"
VERSION="${2#v}"
OUT="$3"
ARCH="${4:-$(uname -m)}"
case "$ARCH" in x86_64) ARCH=amd64 ;; esac
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ICON="$HERE/../icon.png"

# CFBundleVersion must be dotted numerics; dev builds like 2.0.0-dev are not.
PLIST_VERSION="${VERSION%%-*}"

APP="$OUT/kdeps.app"
STAGE="$OUT/dmg-stage"
ICONSET="$OUT/icon.iconset"
rm -rf "$APP" "$STAGE" "$ICONSET"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" "$STAGE" "$ICONSET"

cp "$BIN" "$APP/Contents/MacOS/kdeps-desktop"
sed "s/@VERSION@/$PLIST_VERSION/g" "$HERE/Info.plist.in" > "$APP/Contents/Info.plist"

for s in 16 32 128 256 512; do
  sips -z "$s" "$s" "$ICON" --out "$ICONSET/icon_${s}x${s}.png" >/dev/null
  sips -z $((s * 2)) $((s * 2)) "$ICON" --out "$ICONSET/icon_${s}x${s}@2x.png" >/dev/null
done
iconutil -c icns "$ICONSET" -o "$APP/Contents/Resources/icon.icns"
rm -rf "$ICONSET"

if [ -n "${APPLE_SIGN_IDENTITY:-}" ]; then
  codesign --force --deep --options runtime --timestamp -s "$APPLE_SIGN_IDENTITY" "$APP"
else
  codesign --force --deep -s - "$APP"
fi

cp -R "$APP" "$STAGE/"
ln -s /Applications "$STAGE/Applications"
DMG="$OUT/kdeps-desktop_${VERSION}_darwin_${ARCH}.dmg"
rm -f "$DMG"
# hdiutil create fails intermittently (resource busy / spurious ENOSPC) on busy hosts.
for attempt in 1 2 3 4 5; do
  rm -f "$DMG"
  if hdiutil create -volname "kdeps" -srcfolder "$STAGE" -ov -format UDZO "$DMG" >/dev/null; then
    break
  fi
  [ "$attempt" = 5 ] && { echo "hdiutil create failed" >&2; exit 1; }
  sleep 3
done
rm -rf "$STAGE"
echo "$DMG"
