package librespot

import (
	"context"
	"os"
	"path/filepath"

	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"go.yaml.in/yaml/v3"
)

type LibrespotConfig struct {
	ZeroconfEnabled bool `yaml:"zeroconf_enabled"`
  AudioBackend string `yaml:"audio_backend"`
	DeviceName string `yaml:"device_name"`
  Credentials struct {
    Type string `yaml:"type"`
    SpotifyToken struct {
      Username string `yaml:"username"`
      AccessToken string `yaml:"access_token"`
    } `yaml:"spotify_token"`
  } `yaml:"credentials"`
  Server struct {
    Enabled bool `yaml:"enabled"`
    Address string `yaml:"address"`
    Port int `yaml:"port"`
    AllowOrigin string `yaml:"allow_origin"`
    CertFile string `yaml:"cert_file"`
    KeyFile string `yaml:"key_file"`
    ImageSize string `yaml:"image_size"`
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

func InitLibrespotConfig(ctx context.Context,userId string, accessToken string) error {
	librespotConfig:= makeLibrespotConfig(ctx, userId, accessToken)
	configYaml,err := yaml.Marshal(librespotConfig)
  if err != nil {
    return err
  }
	return os.WriteFile(GetLibrespotConfigFile(), configYaml, 0644)
}

func makeLibrespotConfig(ctx context.Context,userId string, accessToken string) LibrespotConfig{
  var librespotConfig =  LibrespotConfig{}
	librespotConfig.ZeroconfEnabled = utils.GetConfig().Librespot.Daemon.ZeroconfEnabled
	librespotConfig.AudioBackend = getAudioBackend()
	librespotConfig.DeviceName = "lazyspotify"
	librespotConfig.Credentials.Type = "spotify_token"
  librespotConfig.Credentials.SpotifyToken.Username = userId
	librespotConfig.Credentials.SpotifyToken.AccessToken = accessToken
	librespotConfig.Server.Enabled = true
  librespotConfig.Server.Address = utils.GetConfig().Librespot.Host
  librespotConfig.Server.Port = utils.GetConfig().Librespot.Port
  librespotConfig.Server.AllowOrigin = "*"
  librespotConfig.Server.ImageSize = "small"

  return librespotConfig
}

func getAudioBackend() string{
	return  "audio-toolbox"
}
