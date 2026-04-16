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
		Type         string `yaml:"type"`
		SpotifyToken struct {
			Username    string `yaml:"username"`
			AccessToken string `yaml:"access_token"`
		} `yaml:"spotify_token"`
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

func InitLibrespotConfig(ctx context.Context, userId string, accessToken string) error {
	librespotConfig := makeLibrespotConfig(utils.GetConfig(), userId, accessToken)
	configYaml, err := yaml.Marshal(librespotConfig)
	if err != nil {
		return err
	}
	return os.WriteFile(GetLibrespotConfigFile(), configYaml, 0644)
}

func makeLibrespotConfig(cfg utils.AppConfig, userId string, accessToken string) LibrespotConfig {
	var librespotConfig = LibrespotConfig{}

	librespotConfig.LogLevel = daemonLogLevel(cfg.Librespot.Daemon.LogLevel)
	librespotConfig.ZeroconfEnabled = cfg.Librespot.Daemon.ZeroconfEnabled
	librespotConfig.MprisEnabled = mprisEnabledForOS(runtime.GOOS)
	librespotConfig.AudioBackend = getAudioBackend()
	librespotConfig.DeviceName = "lazyspotify"
	librespotConfig.Credentials.Type = "spotify_token"
	librespotConfig.Credentials.SpotifyToken.Username = userId
	librespotConfig.Credentials.SpotifyToken.AccessToken = accessToken
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
	return true
}

func audioBackendForOS(goos string) string {
	switch goos {
	case "darwin":
		return "audio-toolbox"
	default:
		return "alsa"
	}
}
