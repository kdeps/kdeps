#!/usr/bin/env bash
# Usage: make-scoop.sh <version> <windows-zip-sha256>  > kdeps-desktop.json
# Prints the Scoop manifest for the desktop zip attached to release v<version>.
set -euo pipefail

VERSION="${1#v}"
SHA="$2"

cat <<JSON
{
    "version": "${VERSION}",
    "description": "Chat client for the kdeps agent loop (standalone, no kdeps CLI needed)",
    "homepage": "https://kdeps.com/",
    "license": "Apache-2.0",
    "architecture": {
        "64bit": {
            "url": "https://github.com/kdeps/kdeps/releases/download/v${VERSION}/kdeps-desktop_${VERSION}_windows_amd64.zip",
            "hash": "${SHA}"
        }
    },
    "shortcuts": [
        ["kdeps-desktop.exe", "kdeps"]
    ],
    "checkver": "github",
    "autoupdate": {
        "architecture": {
            "64bit": {
                "url": "https://github.com/kdeps/kdeps/releases/download/v\$version/kdeps-desktop_\$version_windows_amd64.zip"
            }
        }
    }
}
JSON
