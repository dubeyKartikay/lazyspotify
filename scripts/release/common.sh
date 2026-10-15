#!/usr/bin/env bash

set -euo pipefail

script_dir() {
  cd "$(dirname "${BASH_SOURCE[0]}")" && pwd
}

readonly REPO_ROOT="${REPO_ROOT:-$(cd "$(script_dir)/../.." && pwd)}"
readonly MANIFEST_PATH="${MANIFEST_PATH:-${REPO_ROOT}/packaging/release-manifest.yaml}"

manifest_value() {
  local key="$1"

  awk -F ':' -v key="$key" '
    $1 == key {
      value = substr($0, index($0, ":") + 1)
      sub(/^[[:space:]]+/, "", value)
      gsub(/^"/, "", value)
      gsub(/"$/, "", value)
      print value
      exit
    }
  ' "${MANIFEST_PATH}"
}

require_command() {
  local command_name="$1"

  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "missing required command: ${command_name}" >&2
    exit 1
  fi
}

release_version() {
  manifest_value "lazyspotify_version" | sed 's/^v//'
}

release_tag() {
  printf 'v%s\n' "$(release_version)"
}

daemon_repo() {
  manifest_value "daemon_repo"
}

daemon_tag() {
  manifest_value "daemon_tag"
}

daemon_commit() {
  manifest_value "daemon_commit"
}

bundle_version() {
  manifest_value "bundle_version"
}

artifact_prefix() {
  printf 'lazyspotify-v%s' "$(release_version)"
}

source_bundle_dirname() {
  printf '%s-src' "$(artifact_prefix)"
}

source_bundle_filename() {
  printf '%s.tar.gz' "$(source_bundle_dirname)"
}

default_dist_dir() {
  printf '%s/dist\n' "${REPO_ROOT}"
}

current_commit() {
  if [[ -n "${SOURCE_COMMIT:-}" ]]; then
    printf '%s\n' "${SOURCE_COMMIT}"
    return
  fi

  if [[ -n "${GITHUB_SHA:-}" ]]; then
    printf '%s\n' "${GITHUB_SHA}"
    return
  fi

  if git -C "${REPO_ROOT}" rev-parse HEAD >/dev/null 2>&1; then
    git -C "${REPO_ROOT}" rev-parse HEAD
    return
  fi

  printf 'unknown\n'
}

build_date_utc() {
  if [[ -n "${BUILD_DATE:-}" ]]; then
    printf '%s\n' "${BUILD_DATE}"
    return
  fi

  date -u +"%Y-%m-%dT%H:%M:%SZ"
}

default_daemon_version() {
  local tag
  tag="$(daemon_tag)"
  if [[ -n "${tag}" ]]; then
    printf '%s\n' "${tag}"
    return
  fi

  local commit
  commit="$(daemon_commit)"
  # go-librespot also sends this as client_version to Spotify's client-token
  # endpoint. An arbitrary label such as daemon-<sha> is rejected with HTTP 400;
  # retain the source revision in the daemon's numeric version format instead.
  printf '0.0.0.g%s\n' "${commit:0:12}"
}

github_repository() {
  if [[ -n "${GITHUB_REPOSITORY:-}" ]]; then
    printf '%s\n' "${GITHUB_REPOSITORY}"
    return
  fi

  if git -C "${REPO_ROOT}" config --get remote.origin.url >/dev/null 2>&1; then
    git -C "${REPO_ROOT}" config --get remote.origin.url | \
      sed -E 's#(git@|https://)([^/:]+)[:/]##; s#\.git$##'
    return
  fi

  printf 'dubeyKartikay/lazyspotify\n'
}

release_asset_url() {
  local asset_name="$1"
  printf 'https://github.com/%s/releases/download/%s/%s\n' \
    "$(github_repository)" \
    "$(release_tag)" \
    "${asset_name}"
}

sha256_file() {
  local path="$1"

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${path}" | awk '{ print $1 }'
    return
  fi

  shasum -a 256 "${path}" | awk '{ print $1 }'
}

lazyspotify_ldflags() {
  local version="$1"
  local commit="$2"
  local build_date="$3"
  local daemon_path="$4"

  printf -- '-X github.com/dubeyKartikay/lazyspotify/buildinfo.Version=%s -X github.com/dubeyKartikay/lazyspotify/buildinfo.Commit=%s -X github.com/dubeyKartikay/lazyspotify/buildinfo.BuildDate=%s -X github.com/dubeyKartikay/lazyspotify/buildinfo.PackagedDaemonPath=%s' \
    "${version}" \
    "${commit}" \
    "${build_date}" \
    "${daemon_path}"
}

load_bundle_metadata() {
  local source_dir="$1"
  local metadata_path="${source_dir}/packaging/source-bundle-metadata.env"

  if [[ -f "${metadata_path}" ]]; then
    # shellcheck disable=SC1090
    source "${metadata_path}"
  fi
}
