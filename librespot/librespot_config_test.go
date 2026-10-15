package librespot

import (
	"runtime"
	"strings"
	"testing"

	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"go.yaml.in/yaml/v3"
)

func TestAudioBackendForOS(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{goos: "darwin", want: "audio-toolbox"},
		{goos: "linux", want: "alsa"},
		{goos: "freebsd", want: "alsa"},
	}

	for _, tt := range tests {
		if got := audioBackendForOS(tt.goos); got != tt.want {
			t.Fatalf("audioBackendForOS(%q) = %q, want %q", tt.goos, got, tt.want)
		}
	}
}

func TestMprisEnabledForOS(t *testing.T) {
	tests := []struct {
		goos string
		want bool
	}{
		{goos: "linux", want: true},
		{goos: "darwin", want: false},
		{goos: "freebsd", want: true},
	}

	for _, tt := range tests {
		if got := mprisEnabledForOS(tt.goos); got != tt.want {
			t.Fatalf("mprisEnabledForOS(%q) = %t, want %t", tt.goos, got, tt.want)
		}
	}
}

func TestMakeLibrespotConfigUsesConfiguredDaemonLogLevel(t *testing.T) {
	cfg := utils.AppConfig{}
	cfg.Librespot.Host = "127.0.0.1"
	cfg.Librespot.Port = 4040
	cfg.Librespot.Daemon.LogLevel = "WARN"

	got := makeLibrespotConfig(cfg)

	if got.LogLevel != "warn" {
		t.Fatalf("makeLibrespotConfig(...).LogLevel = %q, want %q", got.LogLevel, "warn")
	}
}

func TestMakeLibrespotConfigDefaultsDaemonLogLevelToError(t *testing.T) {
	cfg := utils.AppConfig{}
	cfg.Librespot.Host = "127.0.0.1"
	cfg.Librespot.Port = 4040

	got := makeLibrespotConfig(cfg)

	if got.LogLevel != "error" {
		t.Fatalf("makeLibrespotConfig(...).LogLevel = %q, want %q", got.LogLevel, "error")
	}
}

func TestMakeLibrespotConfigEnablesMprisOnLinux(t *testing.T) {
	cfg := utils.AppConfig{}
	cfg.Librespot.Host = "127.0.0.1"
	cfg.Librespot.Port = 4040

	got := makeLibrespotConfig(cfg)

	if runtime.GOOS == "linux" && !got.MprisEnabled {
		t.Fatal("makeLibrespotConfig(...).MprisEnabled = false, want true on linux")
	}

	if runtime.GOOS != "linux" && got.MprisEnabled {
		t.Fatalf("makeLibrespotConfig(...).MprisEnabled = true, want false on %s", runtime.GOOS)
	}
}

func TestMakeLibrespotConfigUsesDeviceAuthCredentials(t *testing.T) {
	got := makeLibrespotConfig(utils.AppConfig{})

	if got.Credentials.Type != "device_auth" {
		t.Fatalf("makeLibrespotConfig(...).Credentials.Type = %q, want %q", got.Credentials.Type, "device_auth")
	}
	if got.ZeroconfEnabled {
		t.Fatal("makeLibrespotConfig(...).ZeroconfEnabled = true, want false")
	}

	encoded, err := yaml.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "access_token") || strings.Contains(string(encoded), "spotify_token") {
		t.Fatal("config contains legacy OAuth credentials")
	}
}
