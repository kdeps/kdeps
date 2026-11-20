#!/usr/bin/env bash
# Usage: make-aur.sh <version> <linux-tarball-sha256> <outdir>
# Writes PKGBUILD and .SRCINFO for kdeps-desktop-bin (release tarball repackaged).
set -euo pipefail

VERSION="${1#v}"
SHA="$2"
OUT="$3"
mkdir -p "$OUT"
URL="https://github.com/kdeps/kdeps/releases/download/v${VERSION}/kdeps-desktop_${VERSION}_linux_amd64.tar.gz"

cat > "$OUT/PKGBUILD" <<PKG
# Maintainer: Joel Bryan Juliano <joelbryan.juliano@gmail.com>
pkgname=kdeps-desktop-bin
pkgver=${VERSION}
pkgrel=1
pkgdesc='Chat client for the kdeps agent loop (standalone, no kdeps CLI needed)'
arch=('x86_64')
url='https://kdeps.com/'
license=('Apache-2.0')
depends=('gtk3' 'webkit2gtk-4.1')
provides=('kdeps-desktop')
conflicts=('kdeps-desktop')
source=("kdeps-desktop_\${pkgver}_linux_amd64.tar.gz::https://github.com/kdeps/kdeps/releases/download/v\${pkgver}/kdeps-desktop_\${pkgver}_linux_amd64.tar.gz")
sha256sums=('${SHA}')

package() {
  install -Dm755 kdeps-desktop "\$pkgdir/usr/bin/kdeps-desktop"
  install -Dm644 kdeps-desktop.desktop "\$pkgdir/usr/share/applications/kdeps-desktop.desktop"
  install -Dm644 kdeps.png "\$pkgdir/usr/share/icons/hicolor/512x512/apps/kdeps.png"
}
PKG

cat > "$OUT/.SRCINFO" <<SRC
pkgbase = kdeps-desktop-bin
	pkgdesc = Chat client for the kdeps agent loop (standalone, no kdeps CLI needed)
	pkgver = ${VERSION}
	pkgrel = 1
	url = https://kdeps.com/
	arch = x86_64
	license = Apache-2.0
	depends = gtk3
	depends = webkit2gtk-4.1
	provides = kdeps-desktop
	conflicts = kdeps-desktop
	source = kdeps-desktop_${VERSION}_linux_amd64.tar.gz::${URL}
	sha256sums = ${SHA}

pkgname = kdeps-desktop-bin
SRC
