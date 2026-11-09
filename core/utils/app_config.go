package utils

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
)

const SpotifyClientIDHelpURL = "https://github.com/dubeyKartikay/lazyspotify?tab=readme-ov-file#set-up-your-spotify-client-id"

const (
	appConfigFileName           = "config.yml"
	spotifyClientIDPlaceholder  = "your_spotify_app_client_id"
	defaultAppConfigFileContent = "auth:\n  client_id: your_spotify_app_client_id\n"
)

// Token store backends select where lazyspotify keeps the Spotify OAuth token.
const (
	TokenStoreBackendKeyring = "keyring"
	TokenStoreBackendFile    = "file"
	TokenStoreBackendCommand = "command"

	defaultTokenStoreCommandTimeout = 10
)

// SupportedTokenStoreBackends lists the accepted auth.token-store.backend
// values, in the order they are documented.
var SupportedTokenStoreBackends = []string{
	TokenStoreBackendKeyring,
	TokenStoreBackendFile,
	TokenStoreBackendCommand,
}

// NormalizeTokenStoreBackend lowercases the configured backend and falls back
// to the system keyring when it is unset.
func NormalizeTokenStoreBackend(backend string) string {
	backend = strings.ToLower(strings.TrimSpace(backend))
	if backend == "" {
		return TokenStoreBackendKeyring
	}
	return backend
}

var (
	config        AppConfig
	configLoadErr error
)

func GetConfig() AppConfig {
	return config
}

func init() {
	config, configLoadErr = LoadConfig()
	if configLoadErr != nil {
		config = getDefaultAppConfig()
	}
}

type AppConfig struct {
	LogLevel string `mapstructure:"log_level"`
	Auth     struct {
		ClientID         string `mapstructure:"client_id"`
		Host             string `mapstructure:"host"`
		Port             int    `mapstructure:"port"`
		RedirectEndpoint string `mapstructure:"redirect-endpoint"`
		Timeout          int    `mapstructure:"timeout"`
		Keyring          struct {
			Service string `mapstructure:"service"`
			Key     string `mapstructure:"key"`
		} `mapstructure:"keyring"`
		TokenStore struct {
			Backend string `mapstructure:"backend"`
			File    struct {
				Path string `mapstructure:"path"`
			} `mapstructure:"file"`
			Command struct {
				Get     []string `mapstructure:"get"`
				Set     []string `mapstructure:"set"`
				Timeout int      `mapstructure:"timeout"`
			} `mapstructure:"command"`
		} `mapstructure:"token-store"`
	} `mapstructure:"auth"`
	Librespot struct {
		Host       string `mapstructure:"host"`
		Port       int    `mapstructure:"port"`
		Timeout    int    `mapstructure:"timeout"`
		RetryDelay int    `mapstructure:"retry-delay"`
		MaxRetries int    `mapstructure:"max-retries"`
		SeekStepMs int    `mapstructure:"seek-step-ms"`
		VolumeStep int    `mapstructure:"volume-step"`
		Daemon     struct {
			Cmd             []string `mapstructure:"cmd"`
			LogLevel        string   `mapstructure:"log_level"`
			ZeroconfEnabled bool     `mapstructure:"zeroconf_enabled"`
		} `mapstructure:"daemon"`
	} `mapstructure:"librespot"`
}

func (c AppConfig) SpotifyClientID() string {
	return strings.TrimSpace(c.Auth.ClientID)
}

func getDefaultAppConfig() AppConfig {
	cfg := AppConfig{}
	cfg.LogLevel = "ERROR"
	cfg.Auth.Host = "127.0.0.1"
	cfg.Auth.Port = 8287
	cfg.Auth.RedirectEndpoint = "/callback"
	cfg.Auth.Timeout = 30
	cfg.Auth.Keyring.Service = "spotify"
	cfg.Auth.Keyring.Key = "token-v2"
	cfg.Auth.TokenStore.Backend = TokenStoreBackendKeyring
	cfg.Auth.TokenStore.Command.Timeout = defaultTokenStoreCommandTimeout
	cfg.Librespot.Host = "127.0.0.1"
	cfg.Librespot.Port = 4040
	cfg.Librespot.Timeout = 180
	cfg.Librespot.RetryDelay = 100
	cfg.Librespot.MaxRetries = 3
	cfg.Librespot.SeekStepMs = 5000
	cfg.Librespot.VolumeStep = 20
	cfg.Librespot.Daemon.LogLevel = "ERROR"
	return cfg
}

func LoadConfig() (AppConfig, error) {
	configDir, err := ensureAppConfigFile()
	if err != nil {
		return AppConfig{}, err
	}

	v := viper.New()
	applyConfigDefaults(v)
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	err = v.ReadInConfig()
	var configErr viper.ConfigFileNotFoundError
	if err != nil && !errors.As(err, &configErr) {
		return AppConfig{}, err
	}
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	var config AppConfig
	err = v.Unmarshal(&config)
	if err != nil {
		return AppConfig{}, err
	}
	return config, nil
}

func ValidateStartupConfig() error {
	if configLoadErr != nil {
		return fmt.Errorf("failed to load config: %w", configLoadErr)
	}
	return validateStartupConfig(config)
}

func validateStartupConfig(cfg AppConfig) error {
	if clientID := cfg.SpotifyClientID(); clientID == "" || clientID == spotifyClientIDPlaceholder {
		return fmt.Errorf("missing required config value `auth.client_id`; see %s", SpotifyClientIDHelpURL)
	}
	return validateTokenStoreConfig(cfg)
}

func validateTokenStoreConfig(cfg AppConfig) error {
	store := cfg.Auth.TokenStore
	switch NormalizeTokenStoreBackend(store.Backend) {
	case TokenStoreBackendKeyring, TokenStoreBackendFile:
		return nil
	case TokenStoreBackendCommand:
		if len(store.Command.Get) == 0 {
			return fmt.Errorf("missing required config value `auth.token-store.command.get` for the command token store")
		}
		if len(store.Command.Set) == 0 {
			return fmt.Errorf("missing required config value `auth.token-store.command.set` for the command token store")
		}
		return nil
	default:
		return fmt.Errorf(
			"unsupported config value `auth.token-store.backend: %s`; supported backends: %s",
			store.Backend,
			strings.Join(SupportedTokenStoreBackends, ", "),
		)
	}
}

func ensureAppConfigFile() (string, error) {
	configDir := getConfigDir()
	if configDir == "" {
		return "", fmt.Errorf("failed to resolve user config directory")
	}
	if err := EnsureExists(configDir); err != nil {
		return "", err
	}

	configPath := filepath.Join(configDir, appConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return configDir, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if err := os.WriteFile(configPath, []byte(defaultAppConfigFileContent), 0644); err != nil {
		return "", err
	}
	return configDir, nil
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

func applyConfigDefaults(v *viper.Viper) {
	defaults := getDefaultAppConfig()
	v.SetDefault("log_level", defaults.LogLevel)
	v.SetDefault("auth.host", defaults.Auth.Host)
	v.SetDefault("auth.port", defaults.Auth.Port)
	v.SetDefault("auth.redirect-endpoint", defaults.Auth.RedirectEndpoint)
	v.SetDefault("auth.timeout", defaults.Auth.Timeout)
	v.SetDefault("auth.keyring.service", defaults.Auth.Keyring.Service)
	v.SetDefault("auth.keyring.key", defaults.Auth.Keyring.Key)
	v.SetDefault("auth.token-store.backend", defaults.Auth.TokenStore.Backend)
	v.SetDefault("auth.token-store.file.path", defaults.Auth.TokenStore.File.Path)
	v.SetDefault("auth.token-store.command.get", defaults.Auth.TokenStore.Command.Get)
	v.SetDefault("auth.token-store.command.set", defaults.Auth.TokenStore.Command.Set)
	v.SetDefault("auth.token-store.command.timeout", defaults.Auth.TokenStore.Command.Timeout)
	v.SetDefault("librespot.host", defaults.Librespot.Host)
	v.SetDefault("librespot.port", defaults.Librespot.Port)
	v.SetDefault("librespot.timeout", defaults.Librespot.Timeout)
	v.SetDefault("librespot.retry-delay", defaults.Librespot.RetryDelay)
	v.SetDefault("librespot.max-retries", defaults.Librespot.MaxRetries)
	v.SetDefault("librespot.seek-step-ms", defaults.Librespot.SeekStepMs)
	v.SetDefault("librespot.volume-step", defaults.Librespot.VolumeStep)
	v.SetDefault("librespot.daemon.log_level", defaults.Librespot.Daemon.LogLevel)
	v.SetDefault("librespot.daemon.zeroconf_enabled", defaults.Librespot.Daemon.ZeroconfEnabled)
}
