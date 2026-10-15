%global debug_package %{nil}
%undefine _debugsource_packages

Name:           lazyspotify
Version:        %{?lazyspotify_version}%{!?lazyspotify_version:0.1.0}
Release:        %{?lazyspotify_release}%{!?lazyspotify_release:1}%{?dist}
Summary:        Terminal Spotify client bundled with a patched go-librespot daemon
License:        MIT AND GPL-3.0-only
URL:            https://github.com/dubeyKartikay/lazyspotify
Source0:        %{name}-v%{version}-src.tar.gz
ExclusiveArch:  x86_64

BuildRequires:  alsa-lib-devel
BuildRequires:  flac-devel
BuildRequires:  gcc
BuildRequires:  go
BuildRequires:  libogg-devel
BuildRequires:  libvorbis-devel
BuildRequires:  mpg123-devel
BuildRequires:  pkgconfig

%description
lazyspotify is a terminal Spotify client that uses a patched go-librespot
daemon for playback.

Linux users need a working Secret Service-compatible keyring at runtime.
Optional clipboard integration is available via wl-clipboard, xclip, or xsel.

%prep
%autosetup -n %{name}-v%{version}-src

%build
mkdir -p target

export CGO_ENABLED=1

go build -trimpath -buildvcs=false \
  -ldflags "-X github.com/dubeyKartikay/lazyspotify/buildinfo.Version=%{version} -X github.com/dubeyKartikay/lazyspotify/buildinfo.Commit=%{?source_commit}%{!?source_commit:unknown} -X github.com/dubeyKartikay/lazyspotify/buildinfo.BuildDate=%{?build_date}%{!?build_date:unknown} -X github.com/dubeyKartikay/lazyspotify/buildinfo.PackagedDaemonPath=%{_libexecdir}/lazyspotify/lazyspotify-librespot" \
  -o target/lazyspotify \
  ./cmd/lazyspotify

pushd third_party/go-librespot
go build -trimpath -buildvcs=false \
  -ldflags "-X github.com/devgianlu/go-librespot.version=%{?daemon_version}%{!?daemon_version:v%{version}}" \
  -o %{_builddir}/%{name}-v%{version}-src/target/lazyspotify-librespot \
  ./cmd/daemon
popd

%install
install -Dm755 target/lazyspotify %{buildroot}%{_bindir}/lazyspotify
install -Dm755 target/lazyspotify-librespot %{buildroot}%{_libexecdir}/lazyspotify/lazyspotify-librespot
install -Dm644 LICENSE %{buildroot}%{_licensedir}/%{name}/LICENSE
install -Dm644 third_party/go-librespot/LICENSE %{buildroot}%{_licensedir}/%{name}/LICENSE.go-librespot
install -Dm644 README.md %{buildroot}%{_docdir}/%{name}/README.md

%check
go test ./...

%files
%license %{_licensedir}/%{name}/LICENSE
%license %{_licensedir}/%{name}/LICENSE.go-librespot
%doc %{_docdir}/%{name}/README.md
%{_bindir}/lazyspotify
%{_libexecdir}/lazyspotify/lazyspotify-librespot

%changelog
* Sun Apr 13 2026 lazyspotify release automation <actions@github.com> - 0.1.0-1
- Initial package
