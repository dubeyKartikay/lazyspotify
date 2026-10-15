# Distribution

`lazyspotify` release automation is driven by
`packaging/release-manifest.yaml`.

## Source of truth

- Git tags `vX.Y.Z` drive GitHub Releases.
- The patched daemon is pinned by `daemon_repo`, `daemon_tag`, and
  `daemon_commit`.
- `scripts/release/create-source-bundle.sh` creates
  `lazyspotify-vX.Y.Z-src.tar.gz`, which vendors the pinned daemon source into
  `third_party/go-librespot` and vendors Go module dependencies for both
  binaries so Launchpad can build without network access.

## Package trees

- Homebrew template: `packaging/homebrew/lazyspotify.rb.tmpl`
- Debian packaging: `debian/`
- RPM spec: `packaging/rpm/lazyspotify.spec`
- AUR metadata: `packaging/aur/lazyspotify-bin/`

## Release helpers

- `scripts/release/build-lazyspotify.sh`: builds the main binary with build
  metadata and a packaged daemon path.
- `scripts/release/build-librespot-daemon.sh`: builds the patched daemon.
- `scripts/release/build-macos-archive.sh`: builds the signed/notarized macOS
  fallback archive when signing env vars are present.
- `scripts/release/build-deb.sh`: builds `.deb` and/or Launchpad source
  packages from the source bundle.
- `scripts/release/build-rpm.sh`: builds `.rpm` and `.src.rpm` from the source
  bundle.
- `scripts/release/build-arch-tarball.sh`: builds the binary tarball consumed
  by the AUR package.
- `scripts/release/render-homebrew-formula.sh`: renders the tap formula from
  the source bundle SHA.
- `scripts/release/render-aur-metadata.sh`: renders `PKGBUILD` and `.SRCINFO`
  from the Arch tarball SHA.

## Workflows

- `.github/workflows/release.yml`: tag-driven end-to-end release pipeline.
- `.github/workflows/build-deb.yml`: manual Debian package build.
- `.github/workflows/build-rpm.yml`: manual RPM package build.
- `.github/workflows/update-homebrew-tap.yml`: manual tap update.
- `.github/workflows/update-aur.yml`: manual AUR update built inside an Arch
  container so `lazyspotify-bin` links against Arch system libraries.

## Release sequence

1. Tag the patched daemon repo and update `packaging/release-manifest.yaml`.
2. Set `lazyspotify_version` and add release notes at `docs/releases/vX.Y.Z.md`.
   Commit and merge the release changes before tagging: the source bundle is
   exported from Git `HEAD`, so uncommitted edits are not included.
3. Tag `lazyspotify` with `vX.Y.Z`. The workflow checks that the tag matches the
   manifest version and that its release notes exist.
4. Let `release.yml` build the source bundle, package artifacts, and draft the
   GitHub Release. Before drafting, it vets and tests both vendored source trees
   with network module downloads disabled, including the daemon's tagged unit
   and integration tests.
5. The workflow updates the Homebrew tap and AUR metadata, uploads the Debian
   source package to Launchpad when the signing secrets are present, and
   submits the Fedora source RPM to COPR when the COPR config is present.
6. The workflow publishes the GitHub Release after those jobs succeed or are
   intentionally skipped.

## Optional publishing configuration

GitHub release assets use the workflow's built-in token. Additional channels
are enabled when their credentials are configured:

- Homebrew: `HOMEBREW_TAP_REPO_TOKEN`.
- AUR: `AUR_SSH_PRIVATE_KEY`.
- macOS signing: `APPLE_DEVELOPER_ID_APP_CERT_B64`,
  `APPLE_DEVELOPER_ID_APP_CERT_PASSWORD`, and
  `APPLE_DEVELOPER_ID_APP_CERT_COMMON_NAME`.
- macOS notarization: `APPLE_NOTARY_APPLE_ID`, `APPLE_NOTARY_TEAM_ID`, and
  `APPLE_NOTARY_APP_PASSWORD`. Without signing/notarization configuration,
  the workflow produces an unsigned macOS archive.
- COPR: `COPR_CONFIG` and `COPR_PROJECT`.

### Launchpad

Launchpad requires `LAUNCHPAD_PPA`, `LAUNCHPAD_GPG_PRIVATE_KEY`, and
`LAUNCHPAD_GPG_KEY_ID`; use `LAUNCHPAD_GPG_PASSPHRASE` for a protected signing key.

- `LAUNCHPAD_PPA` must be set to `owner/archive`, matching the documented
  Launchpad upload target `dput ppa:owner/archive`.
- For this project, use `dubeykartikay/lazyspotify`.
- Setting `LAUNCHPAD_PPA` to only the Launchpad username produces an invalid
  upload target and Launchpad rejects the upload.

## Ubuntu series

- The release workflow currently publishes Launchpad source packages for
  `noble` (Ubuntu 24.04 LTS) and `resolute` (Ubuntu 26.04 LTS).
- Each series is built and uploaded separately so the PPA receives a
  series-specific source version such as `...~noble1` or `...~resolute1`.
- Fedora RPMs are built in a Fedora 44 container.

## Arch packaging

- `lazyspotify-vX.Y.Z-arch-amd64.tar.gz` must be built in an Arch environment.
- Do not reuse the Ubuntu-built Linux tarball for `lazyspotify-bin`; the
  bundled daemon links against distro-specific shared library SONAMEs.
