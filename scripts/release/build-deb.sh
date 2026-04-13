#!/usr/bin/env bash

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

source_bundle=""
output_dir="$(default_dist_dir)"
series="${UBUNTU_SERIES:-noble}"
build_mode="binary"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --source-bundle)
      source_bundle="$2"
      shift 2
      ;;
    --output-dir)
      output_dir="$2"
      shift 2
      ;;
    --series)
      series="$2"
      shift 2
      ;;
    --build-mode)
      build_mode="$2"
      shift 2
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "${source_bundle}" ]]; then
  echo "--source-bundle is required" >&2
  exit 1
fi

require_command tar
require_command dpkg-buildpackage

mkdir -p "${output_dir}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

tar -C "${tmpdir}" -xzf "${source_bundle}"
bundle_root="${tmpdir}/$(source_bundle_dirname)"
build_root="${tmpdir}/lazyspotify-$(release_version)"
orig_tarball="${tmpdir}/lazyspotify_$(release_version).orig.tar.gz"
mkdir -p "${build_root}"
tar -C "${bundle_root}" -cf - . | tar -xf - -C "${build_root}"

# Debian source format 3.0 (quilt) expects an adjacent orig tarball whose top-level
# directory matches the package versioned source tree.
tar -C "${tmpdir}" \
  --exclude="lazyspotify-$(release_version)/debian" \
  -czf "${orig_tarball}" \
  "lazyspotify-$(release_version)"

load_bundle_metadata "${build_root}"

export SOURCE_COMMIT="${LAZYSPOTIFY_COMMIT:-$(current_commit)}"
export BUILD_DATE="${BUILD_DATE:-$(build_date_utc)}"
export DAEMON_VERSION="${DAEMON_VERSION:-${DAEMON_TAG:-$(default_daemon_version)}}"

cat > "${build_root}/debian/changelog" <<EOF
lazyspotify ($(release_version)-1~${series}1) ${series}; urgency=medium

  * Release $(release_version).

 -- lazyspotify release automation <actions@github.com>  $(date -R)
EOF

pushd "${build_root}" >/dev/null
case "${build_mode}" in
  binary)
    dpkg-buildpackage -us -uc -b
    ;;
  source)
    dpkg-buildpackage -us -uc -S -sa
    ;;
  both)
    dpkg-buildpackage -us -uc -b
    dpkg-buildpackage -us -uc -S -sa
    ;;
  *)
    echo "unsupported build mode: ${build_mode}" >&2
    exit 1
    ;;
esac
popd >/dev/null

find "${tmpdir}" -maxdepth 1 \
  \( -name '*.deb' -o -name '*.changes' -o -name '*.buildinfo' -o -name '*.dsc' -o -name '*.tar.*' -o -name '*.xz' \) \
  -exec cp {} "${output_dir}/" \;
