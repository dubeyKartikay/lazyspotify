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
- A working system keyring, or one of the headless token stores described in [Headless and non-keyring setups](#headless-and-non-keyring-setups).
- The patched `lazyspotify-librespot` daemon with device authorization support (only if you are installing from source or running an unpackaged build).
- For macOS source or ZIP installs, install the runtime audio libraries with `brew install flac libogg libvorbis mpg123`. Homebrew package installs handle these dependencies automatically.
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

### Nix

Available in [Nixpkgs](https://search.nixos.org/packages?channel=unstable&query=lazyspotify#show=lazyspotify).

```bash
nix run nixpkgs#lazyspotify
```

### GitHub Releases

Download the latest package from [GitHub Releases](https://github.com/dubeyKartikay/lazyspotify/releases).

- macOS: `.zip`
- Ubuntu/Debian: `.deb`
- Fedora: `.rpm`
- Arch: `.tar.gz`

Example package installs:

```bash
sudo dpkg -i lazyspotify-*.deb
sudo dnf install ./lazyspotify-*.rpm
```

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

## Logging In

Spotify deprecated the token-based login the player daemon previously used, so lazyspotify now signs you in once per browser session:

1. Start `lazyspotify`. On the first run you are asked to log in with your Spotify account through the browser (this uses your own client ID from the section above and powers browsing).
2. After you approve, the same browser page continues straight to Spotify's device pairing screen with the code already filled in. Approve the pairing to connect the player daemon.
3. The daemon stores its own credentials. On later runs lazyspotify starts silently: the login token comes from your configured token store and the player reuses the daemon's stored credentials. You only see the pairing screen again if Spotify invalidated the stored credentials.

The token from your own client ID is only used for Web API requests (playlists, search, library). The player daemon authenticates itself through Spotify's device authorization flow, so it no longer depends on your client ID or token.

### Logging in over SSH

1. Run `lazyspotify` on the remote machine and open its login link in your local browser.
2. Approve the Spotify login. The browser may show a connection error when it redirects to `http://127.0.0.1:8287/callback` on your local machine. Leave that tab open.
3. Copy the **full URL from the address bar**. In lazyspotify, press **p**, paste that URL using your terminal's paste shortcut, and press **Enter**.
4. If the player needs pairing, lazyspotify shows a Spotify pairing link and code. Open that link in the same browser and approve the pairing. The pairing page is hosted by Spotify and works from any machine.


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
| `auth.token-store.key` | No | `token-v2` | Key under which the token is stored, for every backend. |
| `auth.keyring.key` | No | none | Deprecated alias for `auth.token-store.key`. |
| `auth.token-store.backend` | No | `keyring` | Where the login token is stored: `keyring`, `file`, or `command`. See [Headless and non-keyring setups](#headless-and-non-keyring-setups). |
| `auth.token-store.file.path` | No | `token.json` in the config directory | Token file used by the `file` backend. `~` is expanded, relative paths resolve against the config directory, and the path must name a file. |
| `auth.token-store.command.get` | For `command` | none | Argv that prints the stored token on stdout. It must exit with one of `not-found-exit-codes` **only** when no token exists yet. |
| `auth.token-store.command.set` | For `command` | none | Argv that reads a token from stdin and stores it. |
| `auth.token-store.command.not-found-exit-codes` | No | any non-zero exit | Exit codes from `get` that mean "no token stored yet". Set this to protect real failures from being treated as a missing token. |
| `auth.token-store.command.timeout` | No | `10` | Timeout in seconds for each token store command. |

### Librespot Settings

| Key | Required | Default | Notes |
| --- | --- | --- | --- |
| `librespot.host` | No | `127.0.0.1` | Host for the local playback API server. |
| `librespot.port` | No | `4040` | Port for the local playback API server. |
| `librespot.timeout` | No | `180` | Playback API timeout in seconds. |
| `librespot.retry-delay` | No | `100` | Retry delay in milliseconds. |
| `librespot.max-retries` | No | `3` | Retry count for daemon calls. |
| `librespot.seek-step-ms` | No | `5000` | Seek step size in milliseconds. |
| `librespot.volume-step` | No | `20` | Volume step percentage (0-100) used for volume controls. |
| `librespot.daemon.cmd` | Sometimes | none | Required for source/manual installs unless a packaged daemon path was compiled into the binary. |
| `librespot.daemon.log_level` | No | `ERROR` | Log level written into the generated librespot daemon config. |
| `librespot.daemon.zeroconf_enabled` | No | `false` | Enables zeroconf in the daemon config. |

Environment variables can override config values by replacing `.` and `-` with `_`. Examples: `LOG_LEVEL`, `AUTH_CLIENT_ID`, `AUTH_TOKEN_STORE_BACKEND`, `LIBRESPOT_DAEMON_LOG_LEVEL`.

Argv config values (`auth.token-store.command.get`, `auth.token-store.command.set`, `librespot.daemon.cmd`) are shell-split when set from the environment, so spaces separate arguments and quotes keep them together:

```bash
export AUTH_TOKEN_STORE_COMMAND_GET='pass show lazyspotify/token'
export AUTH_TOKEN_STORE_COMMAND_SET='pass insert -m -f "lazyspotify/token"'
```

### Headless and non-keyring setups

The default `keyring` backend needs a system keyring, which on Linux means Secret Service and D-Bus. Headless setups such as WSL instances, containers, and servers often cannot provide one. Set `auth.token-store.backend` to keep the login token somewhere else:

- `keyring` (default): the system keyring.
- `file`: a plaintext JSON file with `0600` permissions. Anyone who can read the file can read your Spotify token. A file that cannot be parsed is moved aside to `<path>.corrupt-<timestamp>` so the next login can rebuild it.
- `command`: an external secret manager such as `pass` or `oo7`. `command.get` prints the token on stdout; `command.set` reads the token from stdin and stores it.

Whatever the backend, a stored value that is not a usable token (`null`, `{}`, no access and no refresh token, or invalid JSON) is treated as missing, so the next login replaces it.

`command.get` must exit non-zero **only** when no token exists yet. By default any non-zero exit means "log in again"; a locked `gpg-agent` or a broken `pass` install would then trigger a login that overwrites a still-valid entry. Set `not-found-exit-codes` to the codes that really mean "missing" so other failures stop the app with the command's stderr instead.

Plaintext file:

```yaml
auth:
  client_id: your_spotify_app_client_id
  token-store:
    backend: file
    file:
      path: ~/.config/lazyspotify/token.json
```

Password Store (`pass`):

```yaml
auth:
  client_id: your_spotify_app_client_id
  token-store:
    backend: command
    command:
      get: [pass, show, lazyspotify/token]
      set: [pass, insert, -m, -f, lazyspotify/token]
      # `pass show` exits 1 when the entry is absent and 2 for GPG failures.
      not-found-exit-codes: [1]
```

The `get` command is expected to fail when no token exists yet, so the first run still shows the login screen. The configured token key (`auth.token-store.key`) is exported to both commands as `LAZYSPOTIFY_TOKEN_KEY`. Changing backends does not migrate existing tokens, so you log in once more after switching.

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
