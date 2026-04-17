{
  description = "Nix packaging for lazyspotify and patched librespot";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
  };

  outputs =
    {
      self,
      nixpkgs,
      ...
    }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system);
    in
    {
      overlays.default =
        final: prev:
        let
          lazyspotify-librespot = final.buildGoModule (finalAttrs: {
            pname = "lazyspotify-librespot";
            version = "0.7.1.1";

            src = final.fetchFromGitHub {
              owner = "dubeyKartikay";
              repo = "go-librespot";
              rev = "v${finalAttrs.version}";
              hash = "sha256-Hq9Qk8f8oKzpBwsbLNAvPO7qam3bh4L4RPUQC67/NZY=";
            };

            buildInputs = [
              final.alsa-lib
              final.flac
              final.libogg
              final.libvorbis
            ];

            nativeBuildInputs = [ final.pkg-config ];

            env.CGO_ENABLED = 1;

            subPackages = [ "cmd/daemon" ];

            ldflags = [
              "-s"
              "-w"
              "-X github.com/devgianlu/go-librespot.version=v${finalAttrs.version}"
            ];

            postInstall = ''
              mv "$out/bin/daemon" "$out/bin/lazyspotify-librespot"
            '';

            vendorHash = "sha256-5J5i2Wc0zHCdvJ3aUkftXeMKS5X8jWimup0Ir4HLuS8=";

            doCheck = false;

            meta = {
              description = "Patched go-librespot daemon for lazyspotify";
              homepage = "https://github.com/dubeyKartikay/go-librespot";
              license = with final.lib.licenses; [ gpl3Only ];
              mainProgram = "lazyspotify-librespot";
              platforms = final.lib.platforms.linux;
              maintainers = [ ];
            };
          });

          lazyspotify = final.buildGoModule (finalAttrs: {
            pname = "lazyspotify";
            version = "0.2.3";

            src = final.fetchFromGitHub {
              owner = "dubeyKartikay";
              repo = "lazyspotify";
              rev = "v${finalAttrs.version}";
              hash = "sha256-JCXKSJmJ3NTwezKDEkb7fdcgbN0Bf1FKRpj7KBQzjMA=";
            };

            buildInputs = [
              final.alsa-lib
              final.flac
              final.libogg
              final.libvorbis
            ];

            nativeBuildInputs = [ final.pkg-config ];

            env.CGO_ENABLED = 1;

            subPackages = [ "cmd/lazyspotify" ];

            ldflags = [
              "-s"
              "-w"
              "-X github.com/dubeyKartikay/lazyspotify/buildinfo.Version=${finalAttrs.version}"
              "-X github.com/dubeyKartikay/lazyspotify/buildinfo.Commit=${finalAttrs.src.rev}"
              "-X github.com/dubeyKartikay/lazyspotify/buildinfo.BuildDate=unknown"
              "-X github.com/dubeyKartikay/lazyspotify/buildinfo.PackagedDaemonPath=${final.lib.getExe lazyspotify-librespot}"
            ];

            checkFlags = [ "./..." ];

            vendorHash = "sha256-Axdt3/3ZOZY9Z5VUI6Wh77oIREOO26ODMyEgtscTmn8=";

            meta = {
              description = "Terminal Spotify client";
              homepage = "https://github.com/dubeyKartikay/lazyspotify";
              license = with final.lib.licenses; [ mit ];
              mainProgram = "lazyspotify";
              platforms = final.lib.platforms.linux;
              maintainers = [ ];
            };
          });
        in
        {
          inherit lazyspotify lazyspotify-librespot;
        };

      packages = forAllSystems (
        system:
        let
          pkgs = import nixpkgs {
            inherit system;
            overlays = [ self.overlays.default ];
          };
        in
        {
          lazyspotify-librespot = pkgs.lazyspotify-librespot;
          lazyspotify = pkgs.lazyspotify;
          default = pkgs.lazyspotify;
        }
      );

      apps = forAllSystems (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.lazyspotify}/bin/lazyspotify";
        };
      });
    };
}
