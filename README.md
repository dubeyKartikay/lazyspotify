<div align="center">
  <img width="220" src="./docs/assets/logo.png" alt="lazyspotify logo" />
  <h1>lazyspotify</h1>
  <p>A terminal Spotify client.</p>
  <p>
    <a href="https://github.com/dubeyKartikay/lazyspotify/releases"><img src="https://img.shields.io/github/v/release/dubeyKartikay/lazyspotify?label=release" alt="Release"/></a>
    <a href="https://github.com/dubeyKartikay/lazyspotify/actions/workflows/release.yml"><img src="https://github.com/dubeyKartikay/lazyspotify/actions/workflows/release.yml/badge.svg" alt="Release workflow"/></a>
    <a href="./LICENSE"><img src="https://img.shields.io/github/license/dubeyKartikay/lazyspotify" alt="License"/></a>
  </p>
</div>


![LazySpotify](docs/assets/lazyspotify-start.png)

## Requirements

- A Spotify Premium account.
- A working system keyring.
- The patched `lazyspotify-librespot` daemon (only if you are installing from source or running an unpackaged build)
- On Linux, one of `wl-clipboard`, `xclip`, or `xsel` if you want clipboard support on the auth screen.

## Install

### Homebrew

```bash
brew tap dubeyKartikay/lazyspotify
brew install lazyspotify
```

### Arch Linux

```bash
yay -S lazyspotify-bin
```

### GitHub Releases

Download the latest package from [GitHub Releases](https://github.com/dubeyKartikay/lazyspotify/releases).

- macOS: signed `.zip`
- Ubuntu/Debian: `.deb`
- Fedora/RHEL: `.rpm`
- Arch: `.tar.gz`

Example package installs:

```bash
sudo dpkg -i lazyspotify-*.deb
sudo dnf install ./lazyspotify-*.rpm
```

### Other Installation Methods

<details>
  <summary><b>Nix / NixOS</b></summary>

  ##### Ephemeral / One-Off Run (No Installation)
  ```bash
  nix run github:dubeyKartikay/lazyspotify
  ```
  ##### Imperative Installation
  ```bash
  nix profile install github:dubeyKartikay/lazyspotify
  ```
  ##### Declarative Installation
  ###### in your flake.nix
  ```nix
  lazyspotify.url = "github:dubeyKartikay/lazyspotify";
  ```
  ###### in your system or home-manager packages
  ```nix
  inputs.lazyspotify.packages.${pkgs.system}.default
  ```
</details>

### Build From Source

Build the app:

```bash
git clone https://github.com/dubeyKartikay/lazyspotify.git
cd lazyspotify
make build
```

Then build the patched daemon from [`dubeyKartikay/go-librespot`](https://github.com/dubeyKartikay/go-librespot) and point `librespot.daemon.cmd` at that binary in your config.

If you build `lazyspotify` yourself and do not compile in a packaged daemon path, `librespot.daemon.cmd` is required.

## Demos

### Play A Track From Playlist > Track

![Play a track from a playlist](docs/assets/demos/playlist-track-play.gif)

### Player Controls

![Player controls](docs/assets/demos/player-controls.gif)

### Library Navigation

![Library navigation](docs/assets/demos/library-navigation.gif)

### Search

![Search navigation](docs/assets/demos/search-navigation.gif)

## Set Up Your Spotify Client ID

`lazyspotify` requires your own Spotify app client ID.

1. Open the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard).
2. Create a new app with:
   - App name: `lazyspotify`
   - App description: `terminal based spotify client`
   - Website: `https://github.com/dubeyKartikay/lazyspotify`
   - Redirect URIs: `http://127.0.0.1:8287/callback`
   - APIs used: `Web API`, `Web Playback SDK`
3. Copy the app's Client ID.
4. Put the Client ID in `config.yml` or export it as an environment variable. See [Configuration](https://github.com/dubeyKartikay/lazyspotify?tab=readme-ov-file#configuration).

Minimal config:

```yaml
auth:
  client_id: your_spotify_app_client_id
```

Environment override:

```bash
export AUTH_CLIENT_ID=your_spotify_app_client_id
```

If you change `auth.host`, `auth.port`, or `auth.redirect-endpoint`, update the Spotify app Redirect URI to match exactly.

## Configuration

Config file locations:

- macOS: `~/Library/Application Support/lazyspotify/config.yml`
- Linux: `~/.config/lazyspotify/config.yml`

If the file is missing, lazyspotify creates `config.yml` with:

```yaml
auth:
  client_id: your_spotify_app_client_id
```

Then replace `your_spotify_app_client_id` with your Spotify app Client ID.

Minimal config for package installs:

```yaml
auth:
  client_id: your_spotify_app_client_id
```

Minimal config for source or manual installs:

```yaml
auth:
  client_id: your_spotify_app_client_id

librespot:
  daemon:
    cmd:
      - /absolute/path/to/lazyspotify-librespot
```

The generated daemon config is written automatically under the `librespot/` subdirectory inside the app config directory. You usually do not need to edit it manually.

### Auth Settings

| Key | Required | Default | Notes |
| --- | --- | --- | --- |
| `log_level` | No | `ERROR` | App log level for `lazyspotify`. |
| `auth.client_id` | Yes | none | Your Spotify app client ID. |
| `auth.host` | No | `127.0.0.1` | Host used for the local OAuth callback server. |
| `auth.port` | No | `8287` | Port used for the local OAuth callback server. |
| `auth.redirect-endpoint` | No | `/callback` | Callback path for Spotify OAuth. |
| `auth.timeout` | No | `30` | Auth server shutdown timeout in seconds. |
| `auth.keyring.service` | No | `spotify` | Keyring service name for stored tokens. |
| `auth.keyring.key` | No | `token-v2` | Keyring key for stored tokens. |

### Librespot Settings

| Key | Required | Default | Notes |
| --- | --- | --- | --- |
| `librespot.host` | No | `127.0.0.1` | Host for the local playback API server. |
| `librespot.port` | No | `4040` | Port for the local playback API server. |
| `librespot.timeout` | No | `180` | Playback API timeout in seconds. |
| `librespot.retry-delay` | No | `100` | Retry delay in milliseconds. |
| `librespot.max-retries` | No | `3` | Retry count for daemon calls. |
| `librespot.seek-step-ms` | No | `5000` | Seek step size in milliseconds. |
| `librespot.volume-step` | No | `3276` | Volume step used for volume controls. |
| `librespot.daemon.cmd` | Sometimes | none | Required for source/manual installs unless a packaged daemon path was compiled into the binary. |
| `librespot.daemon.log_level` | No | `ERROR` | Log level written into the generated librespot daemon config. |
| `librespot.daemon.zeroconf_enabled` | No | `false` | Enables zeroconf in the daemon config. |

Environment variables can override config values by replacing `.` and `-` with `_`. Examples: `LOG_LEVEL`, `AUTH_CLIENT_ID`, `LIBRESPOT_DAEMON_LOG_LEVEL`.

## Run

Start the app with:

```bash
lazyspotify
```

If you built from source:

```bash
./target/lazyspotify
```

Print build metadata:

```bash
lazyspotify version
```

## Development

```bash
make run
go test ./...
```

## Community

- [Code of Conduct](./.github/CODE_OF_CONDUCT.md)
- [Contributing Guide](./.github/CONTRIBUTING.md)
- [Security Policy](./.github/SECURITY.md)
