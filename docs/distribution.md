# Distribution

`lazyspotify` release automation is driven by
`packaging/release-manifest.yaml`.

## Source of truth

- Git tags `vX.Y.Z` drive GitHub Releases.
- The patched daemon is pinned by `daemon_repo`, `daemon_tag`, and
  `daemon_commit`.
- `scripts/release/create-source-bundle.sh` creates
  `lazyspotify-vX.Y.Z-src.tar.gz`, which vendors the pinned daemon source into
  `third_party/go-librespot`.

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
2. Tag `lazyspotify` with `vX.Y.Z`.
3. Let `release.yml` build the source bundle, package artifacts, and draft the
   GitHub Release.
4. The workflow updates the Homebrew tap and AUR metadata, uploads the Debian
   source package to Launchpad when the signing secrets are present, and
   submits the Fedora source RPM to COPR when the COPR config is present.
5. The workflow publishes the GitHub Release after those jobs succeed or are
   intentionally skipped.

## Launchpad secrets

- `LAUNCHPAD_PPA` must be set to `owner/archive`, matching the documented
  Launchpad upload target `dput ppa:owner/archive`.
- For this project, use `dubeykartikay/lazyspotify`.
- Setting `LAUNCHPAD_PPA` to only the Launchpad username produces an invalid
  upload target and Launchpad rejects the upload.

## Ubuntu series

- The release workflow currently publishes Launchpad source packages for
  `noble` (Ubuntu 24.04 LTS) and `questing` (Ubuntu 25.10).
- Each series is built and uploaded separately so the PPA receives a
  series-specific source version such as `...~noble1` or `...~questing1`.

## Arch packaging

- `lazyspotify-vX.Y.Z-arch-amd64.tar.gz` must be built in an Arch environment.
- Do not reuse the Ubuntu-built Linux tarball for `lazyspotify-bin`; the
  bundled daemon links against distro-specific shared library SONAMEs.
