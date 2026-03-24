package utils

import (
	"os"
	"path/filepath"
	"strings"
	"github.com/spf13/viper"
)

var config AppConfig

func GetConfig() AppConfig {
	return config
}

func init() {
	var err error
	config, err = LoadConfig()
	if err != nil {
		config = getDefaultAppConfig()
	}
}

type AppConfig struct{
	SpotifyClientId string `mapstructure:"spotify-client-id"`
	Auth struct{
    Host string `mapstructure:"host"`
    Port int `mapstructure:"port"`
		RedirectEndpoint string `mapstructure:"redirect-endpoint"`
		Timeout int `mapstructure:"timeout"`
		Keyring struct{
      Service string `mapstructure:"service"`
			Key string `mapstructure:"key"`
		} `mapstructure:"keyring"`
	} `mapstructure:"auth"`
  Librespot struct{
    Host string `mapstructure:"host"`
    Port int `mapstructure:"port"`
		Timeout int `mapstructure:"timeout"`
		RetryDelay int `mapstructure:"retry-delay"`
    MaxRetries int `mapstructure:"max-retries"`
		Daemon struct{
      Cmd []string `mapstructure:"cmd"`
			ZeroconfEnabled bool `mapstructure:"zeroconf_enabled"`
		} `mapstructure:"daemon"`

  } `mapstructure:"librespot"`
}

func getDefaultAppConfig() AppConfig {
	cfg := AppConfig{}
	cfg.Auth.Host = "127.0.0.1"
	cfg.Auth.Port = 8287
	cfg.Auth.RedirectEndpoint = "/callback"
	cfg.Auth.Timeout = 30
	cfg.Auth.Keyring.Service = "spotify"
	cfg.Auth.Keyring.Key = "token-v2"
	cfg.Librespot.Host = "127.0.0.1"
	cfg.Librespot.Port = 4040
	cfg.Librespot.Timeout = 180
	cfg.Librespot.RetryDelay = 100
	cfg.Librespot.MaxRetries = 3
	cfg.Librespot.Daemon.Cmd = []string{"/Users/user/personal/go-librespot/daemon"}
	return cfg
}

func LoadConfig() (AppConfig, error) {
	v := viper.New()
  v.SetConfigName("config")
  v.SetConfigType("yaml")
	v.AddConfigPath(SafeGetConfigDir())
  err := v.ReadInConfig()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
  if err != nil {
    return AppConfig{}, err
  }
  var config AppConfig
  err = v.Unmarshal(&config)
  if err != nil {
    return AppConfig{}, err
  }
  return config, nil
}

func getConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	configDir := filepath.Join(dir, "lazyspotify")
  return configDir
}

func SafeGetConfigDir() string {
  configDir := getConfigDir()
	EnsureExists(configDir)
  return configDir
}


