#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

source_dir="${REPO_ROOT}"
daemon_source_dir=""
output_dir="$(default_dist_dir)"
arch=""
version="$(release_version)"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --source-dir)
      source_dir="$2"
      shift 2
      ;;
    --daemon-source-dir)
      daemon_source_dir="$2"
      shift 2
      ;;
    --output-dir)
      output_dir="$2"
      shift 2
      ;;
    --arch)
      arch="$2"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "${arch}" ]]; then
  echo "--arch is required" >&2
  exit 1
fi

if [[ -z "${daemon_source_dir}" ]]; then
  daemon_source_dir="${source_dir}/third_party/go-librespot"
fi

case "${arch}" in
  amd64)
    goarch="amd64"
    ;;
  arm64)
    goarch="arm64"
    ;;
  *)
    echo "unsupported macOS arch: ${arch}" >&2
    exit 1
    ;;
esac

mkdir -p "${output_dir}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

artifact_dir="${tmpdir}/$(artifact_prefix)-macos-${arch}"
GOOS=darwin GOARCH="${goarch}" \
  "$(dirname "${BASH_SOURCE[0]}")/build-lazyspotify.sh" \
  --source-dir "${source_dir}" \
  --output "${artifact_dir}/lazyspotify" \
  --daemon-path ""

GOOS=darwin GOARCH="${goarch}" \
  "$(dirname "${BASH_SOURCE[0]}")/build-librespot-daemon.sh" \
    --source-dir "${daemon_source_dir}" \
    --output "${artifact_dir}/lazyspotify-librespot"

cp "${source_dir}/LICENSE" "${artifact_dir}/LICENSE"
cp "${daemon_source_dir}/LICENSE" "${artifact_dir}/LICENSE.go-librespot"
cp "${source_dir}/README.md" "${artifact_dir}/README.md"

cat > "${artifact_dir}/INSTALL.txt" <<EOF
Runtime audio libraries are required. With Homebrew, install them using:
  brew install flac libogg libvorbis mpg123

This archive ships both binaries, but it does not install them into a fixed
system path. After extracting the archive, set librespot.daemon.cmd in your
lazyspotify config so the app can find lazyspotify-librespot.
EOF

if [[ -n "${MACOS_CODESIGN_IDENTITY:-}" ]]; then
  codesign --force --options runtime --timestamp --sign "${MACOS_CODESIGN_IDENTITY}" "${artifact_dir}/lazyspotify"
  codesign --force --options runtime --timestamp --sign "${MACOS_CODESIGN_IDENTITY}" "${artifact_dir}/lazyspotify-librespot"
fi

archive_path="${output_dir}/$(artifact_prefix)-macos-${arch}.zip"
ditto -c -k --keepParent "${artifact_dir}" "${archive_path}"

if [[ -n "${MACOS_NOTARY_PROFILE:-}" ]]; then
  xcrun notarytool submit "${archive_path}" --keychain-profile "${MACOS_NOTARY_PROFILE}" --wait
fi

printf '%s\n' "${archive_path}"
