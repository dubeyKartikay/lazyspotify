#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

archive=""
pkgbuild_output=""
srcinfo_output=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --archive)
      archive="$2"
      shift 2
      ;;
    --pkgbuild-output)
      pkgbuild_output="$2"
      shift 2
      ;;
    --srcinfo-output)
      srcinfo_output="$2"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "${archive}" || -z "${pkgbuild_output}" || -z "${srcinfo_output}" ]]; then
  echo "--archive, --pkgbuild-output, and --srcinfo-output are required" >&2
  exit 1
fi

archive_sha="$(sha256_file "${archive}")"
archive_url="$(release_asset_url "$(artifact_prefix)-arch-amd64.tar.gz")"

mkdir -p "$(dirname "${pkgbuild_output}")"

cat > "${pkgbuild_output}" <<EOF
pkgname=lazyspotify-bin
pkgver=$(release_version)
pkgrel=1
pkgdesc="Terminal Spotify client bundled with a patched go-librespot daemon"
arch=('x86_64')
url="https://github.com/$(github_repository)"
license=('MIT' 'GPL3')
depends=('alsa-lib' 'flac' 'glibc' 'libogg' 'libvorbis' 'mpg123')
optdepends=(
  'gnome-keyring: Secret Service keyring provider'
  'keepassxc: Secret Service keyring provider'
  'kwallet: Secret Service keyring provider'
  'wl-clipboard: clipboard integration on Wayland'
  'xclip: clipboard integration on X11'
  'xsel: clipboard integration on X11'
)
source=("lazyspotify-v\${pkgver}-arch-amd64.tar.gz::${archive_url}")
sha256sums=('${archive_sha}')

package() {
  install -Dm755 "\${srcdir}/lazyspotify-v\${pkgver}-arch-amd64/lazyspotify" "\${pkgdir}/usr/bin/lazyspotify"
  install -Dm755 "\${srcdir}/lazyspotify-v\${pkgver}-arch-amd64/lazyspotify-librespot" "\${pkgdir}/usr/lib/lazyspotify/lazyspotify-librespot"
  install -Dm644 "\${srcdir}/lazyspotify-v\${pkgver}-arch-amd64/LICENSE" "\${pkgdir}/usr/share/licenses/\${pkgname}/LICENSE"
  install -Dm644 "\${srcdir}/lazyspotify-v\${pkgver}-arch-amd64/LICENSE.go-librespot" "\${pkgdir}/usr/share/licenses/\${pkgname}/LICENSE.go-librespot"
  install -Dm644 "\${srcdir}/lazyspotify-v\${pkgver}-arch-amd64/README.md" "\${pkgdir}/usr/share/doc/\${pkgname}/README.md"
}
EOF

cat > "${srcinfo_output}" <<EOF
pkgbase = lazyspotify-bin
	pkgdesc = Terminal Spotify client bundled with a patched go-librespot daemon
	pkgver = $(release_version)
	pkgrel = 1
	url = https://github.com/$(github_repository)
	arch = x86_64
	license = MIT
	license = GPL3
	depends = alsa-lib
	depends = flac
	depends = glibc
	depends = libogg
	depends = libvorbis
	depends = mpg123
	optdepends = gnome-keyring: Secret Service keyring provider
	optdepends = keepassxc: Secret Service keyring provider
	optdepends = kwallet: Secret Service keyring provider
	optdepends = wl-clipboard: clipboard integration on Wayland
	optdepends = xclip: clipboard integration on X11
	optdepends = xsel: clipboard integration on X11
	source = $(artifact_prefix)-arch-amd64.tar.gz::${archive_url}
	sha256sums = ${archive_sha}

pkgname = lazyspotify-bin
EOF
