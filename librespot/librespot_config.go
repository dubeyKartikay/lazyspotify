package librespot

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"go.yaml.in/yaml/v3"
)

type LibrespotConfig struct {
	LogLevel        string `yaml:"log_level"`
	ZeroconfEnabled bool   `yaml:"zeroconf_enabled"`
	MprisEnabled    bool   `yaml:"mpris_enabled"`
	AudioBackend    string `yaml:"audio_backend"`
	DeviceName      string `yaml:"device_name"`
	Credentials     struct {
		Type     string `yaml:"type"`
		Zeroconf struct {
			PersistCredentials bool `yaml:"persist_credentials"`
		} `yaml:"zeroconf"`
	} `yaml:"credentials"`
	Server struct {
		Enabled     bool   `yaml:"enabled"`
		Address     string `yaml:"address"`
		Port        int    `yaml:"port"`
		AllowOrigin string `yaml:"allow_origin"`
		CertFile    string `yaml:"cert_file"`
		KeyFile     string `yaml:"key_file"`
		ImageSize   string `yaml:"image_size"`
	} `yaml:"server"`
}

func GetLibrespotConfigDir() string {
	configDir := utils.SafeGetConfigDir()
	librespotConfigDir := filepath.Join(configDir, "librespot")
	utils.EnsureExists(librespotConfigDir)
	return librespotConfigDir
}

func GetLibrespotConfigFile() string {
	return filepath.Join(GetLibrespotConfigDir(), "config.yml")
}

func InitLibrespotConfig(ctx context.Context) error {
	librespotConfig := makeLibrespotConfig(utils.GetConfig())
	configYaml, err := yaml.Marshal(librespotConfig)
	if err != nil {
		return err
	}
	return os.WriteFile(GetLibrespotConfigFile(), configYaml, 0644)
}

func makeLibrespotConfig(cfg utils.AppConfig) LibrespotConfig {
	var librespotConfig = LibrespotConfig{}

	librespotConfig.LogLevel = daemonLogLevel(cfg.Librespot.Daemon.LogLevel)

	// Zeroconf is required for authentication.
	librespotConfig.ZeroconfEnabled = true
	librespotConfig.Credentials.Type = "zeroconf"
	librespotConfig.Credentials.Zeroconf.PersistCredentials = true
	librespotConfig.MprisEnabled = mprisEnabledForOS(runtime.GOOS)
	librespotConfig.AudioBackend = getAudioBackend()
	librespotConfig.DeviceName = "lazyspotify"

	// Spotify Login5 no longer accepts the OAuth token used by LazySpotify.
	// Authentication is handled by go-librespot's Zeroconf flow instead.
	librespotConfig.Server.Enabled = true
	librespotConfig.Server.Address = cfg.Librespot.Host
	librespotConfig.Server.Port = cfg.Librespot.Port
	librespotConfig.Server.AllowOrigin = "*"
	librespotConfig.Server.ImageSize = "small"

	return librespotConfig
}

func daemonLogLevel(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return "error"
	}
	return normalized
}

func getAudioBackend() string {
	return audioBackendForOS(runtime.GOOS)
}

func mprisEnabledForOS(goos string) bool {
	return goos != "darwin"
}

func audioBackendForOS(goos string) string {
	switch goos {
	case "darwin":
		return "audio-toolbox"
	default:
		return "alsa"
	}
}
