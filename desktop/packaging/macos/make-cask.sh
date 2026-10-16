#!/usr/bin/env bash
# Usage: make-cask.sh <version> <arm64-sha256> <amd64-sha256>  > kdeps-desktop.rb
# Prints the Homebrew cask for the desktop dmgs attached to release v<version>.
set -euo pipefail

VERSION="${1#v}"
ARM_SHA="$2"
INTEL_SHA="$3"

cat <<CASK
cask "kdeps-desktop" do
  version "${VERSION}"

  on_arm do
    sha256 "${ARM_SHA}"

    url "https://github.com/kdeps/kdeps/releases/download/v#{version}/kdeps-desktop_#{version}_darwin_arm64.dmg"
  end
  on_intel do
    sha256 "${INTEL_SHA}"

    url "https://github.com/kdeps/kdeps/releases/download/v#{version}/kdeps-desktop_#{version}_darwin_amd64.dmg"
  end

  name "kdeps"
  desc "Desktop chat client for the kdeps agent loop"
  homepage "https://kdeps.com/"

  livecheck do
    skip "Auto-generated on release."
  end

  depends_on :macos

  app "kdeps.app"

  # The app is ad-hoc signed, not notarized: clear the quarantine flag so it opens.
  postflight_steps do
    run "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "{{appdir}}/kdeps.app"], must_succeed: false
  end

  zap trash: "~/.kdeps"
end
CASK
