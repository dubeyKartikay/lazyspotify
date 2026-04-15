# Contributing to lazyspotify

Thanks for contributing to `lazyspotify`.

## Before You Start

- Read the [Code of Conduct](./CODE_OF_CONDUCT.md).
- Search existing issues and pull requests before opening a new one.
- Keep changes focused. Small, reviewable pull requests are easier to merge and
  less likely to regress terminal behavior.

## Development Setup

### Requirements

- Go `1.25.6` or newer.
- A Spotify Premium account if you want to exercise playback flows manually.
- A working system keyring.
- The patched `lazyspotify-librespot` daemon for source builds.

### Clone and Run

```bash
git clone https://github.com/dubeyKartikay/lazyspotify.git
cd lazyspotify
make run
```

### Build

```bash
make build
```

The binary is written to `target/lazyspotify`.

### Patched Daemon for Source Builds

If you are running a manually built `lazyspotify` binary, make sure
`librespot.daemon.cmd` points to a working `lazyspotify-librespot` binary.

If you already have a packaged install, you can reuse its daemon path:

```bash
lazyspotify version
```

Look for:

```text
packaged_daemon_path=/path/to/lazyspotify-librespot
```

Save that path and add it to your config:

- macOS config directory: `~/Library/Application Support/lazyspotify/`
- Linux config directory: `~/.config/lazyspotify/`

```yaml
librespot:
  daemon:
    cmd:
      - "/path/to/lazyspotify-librespot"
```

If you do not have a packaged daemon available, build the patched daemon from
[`dubeyKartikay/go-librespot`](https://github.com/dubeyKartikay/go-librespot):

```bash
git clone https://github.com/dubeyKartikay/go-librespot.git
cd go-librespot
go build -o lazyspotify-librespot ./cmd/daemon
```

Then point `librespot.daemon.cmd` at the absolute path to that
`lazyspotify-librespot` binary.

For a full manual source-build flow:

```bash
git clone https://github.com/dubeyKartikay/lazyspotify.git
cd lazyspotify
make build
```

This produces `target/lazyspotify`. Once `librespot.daemon.cmd` is configured,
run the compiled binary directly:

```bash
./target/lazyspotify
```

## Testing

Run the project test suite before opening a pull request:

```bash
go test ./...
```

If your change affects packaging, authentication, or playback flows, include the
manual verification steps you ran in the pull request description.

## Reporting Issues

Use the issue templates whenever they fit:

- Bug reports should include steps to reproduce, expected behavior, actual
  behavior, and environment details.
- Feature requests should explain the problem being solved, not only the desired
  UI or implementation.

For security issues, do not open a public issue. Follow the
[Security Policy](./SECURITY.md) instead.

## Submitting Pull Requests

1. Fork the repository and create a topic branch from `main`.
2. Make your change with clear commit messages.
3. Run `go test ./...`.
4. Update documentation when behavior, configuration, or installation changes.
5. Open a pull request using the provided template.

## Pull Request Expectations

- Explain the user-facing change clearly.
- Keep unrelated refactors out of the same PR.
- Add or update tests when practical.
- Call out breaking changes, packaging changes, or follow-up work explicitly.

## Style Notes

- Follow the existing Go code style and project layout.
- Prefer simple implementations over broad refactors.
- Preserve compatibility with the documented configuration keys unless the change
  intentionally introduces a migration.
