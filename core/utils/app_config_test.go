package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppConfigSpotifyClientIDPrefersAuthClientID(t *testing.T) {
	cfg := AppConfig{}
	cfg.Auth.ClientID = "auth-client-id"

	if got := cfg.SpotifyClientID(); got != "auth-client-id" {
		t.Fatalf("SpotifyClientID() = %q, want %q", got, "auth-client-id")
	}
}

func TestAppConfigSpotifyClientIDTrimsWhitespace(t *testing.T) {
	cfg := AppConfig{}
	cfg.Auth.ClientID = " auth-client-id "

	if got := cfg.SpotifyClientID(); got != "auth-client-id" {
		t.Fatalf("SpotifyClientID() = %q, want %q", got, "auth-client-id")
	}
}

func TestValidateStartupConfigRequiresSpotifyClientID(t *testing.T) {
	err := validateStartupConfig(AppConfig{})
	if err == nil {
		t.Fatal("validateStartupConfig() returned nil, want error")
	}
	want := "missing required config value `auth.client_id`"
	if got := err.Error(); !strings.HasPrefix(got, want) {
		t.Fatalf("validateStartupConfig() error = %q, want prefix %q", got, want)
	}
}

func TestValidateStartupConfigAcceptsConfiguredSpotifyClientID(t *testing.T) {
	cfg := AppConfig{}
	cfg.Auth.ClientID = "configured-client-id"

	if err := validateStartupConfig(cfg); err != nil {
		t.Fatalf("validateStartupConfig() error = %v, want nil", err)
	}
}

func TestValidateStartupConfigRejectsPlaceholderSpotifyClientID(t *testing.T) {
	cfg := AppConfig{}
	cfg.Auth.ClientID = spotifyClientIDPlaceholder

	err := validateStartupConfig(cfg)
	if err == nil {
		t.Fatal("validateStartupConfig() returned nil, want error")
	}
	want := "missing required config value `auth.client_id`"
	if got := err.Error(); !strings.HasPrefix(got, want) {
		t.Fatalf("validateStartupConfig() error = %q, want prefix %q", got, want)
	}
}

func TestGetDefaultAppConfigUsesErrorLogLevels(t *testing.T) {
	cfg := getDefaultAppConfig()

	if got := cfg.LogLevel; got != "ERROR" {
		t.Fatalf("getDefaultAppConfig().LogLevel = %q, want %q", got, "ERROR")
	}
	if got := cfg.Librespot.Daemon.LogLevel; got != "ERROR" {
		t.Fatalf("getDefaultAppConfig().Librespot.Daemon.LogLevel = %q, want %q", got, "ERROR")
	}
}

func TestGetDefaultAppConfigVolumeStepIsPercentage(t *testing.T) {
	cfg := getDefaultAppConfig()

	if got := cfg.Librespot.VolumeStep; got != 20 {
		t.Fatalf("getDefaultAppConfig().Librespot.VolumeStep = %d, want 20", got)
	}
}

func TestLoadConfigCreatesConfigYMLWhenMissing(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}

	configPath := filepath.Join(getConfigDir(), appConfigFileName)
	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", configPath, err)
	}
	if string(got) != defaultAppConfigFileContent {
		t.Fatalf("config file contents = %q, want %q", string(got), defaultAppConfigFileContent)
	}
	if cfg.Auth.ClientID != spotifyClientIDPlaceholder {
		t.Fatalf("LoadConfig().Auth.ClientID = %q, want %q", cfg.Auth.ClientID, spotifyClientIDPlaceholder)
	}
}

func TestLoadConfigUsesExistingConfigYML(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	configDir := getConfigDir()
	if err := EnsureExists(configDir); err != nil {
		t.Fatalf("EnsureExists(%q) error = %v, want nil", configDir, err)
	}

	configPath := filepath.Join(configDir, appConfigFileName)
	wantConfig := "auth:\n  client_id: existing-client-id\n"
	if err := os.WriteFile(configPath, []byte(wantConfig), 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", configPath, err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	if cfg.Auth.ClientID != "existing-client-id" {
		t.Fatalf("LoadConfig().Auth.ClientID = %q, want %q", cfg.Auth.ClientID, "existing-client-id")
	}

	got, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", configPath, err)
	}
	if string(got) != wantConfig {
		t.Fatalf("config file contents = %q, want %q", string(got), wantConfig)
	}
}

func TestNormalizeTokenStoreBackend(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: TokenStoreBackendKeyring},
		{in: "  ", want: TokenStoreBackendKeyring},
		{in: "keyring", want: TokenStoreBackendKeyring},
		{in: "FILE", want: TokenStoreBackendFile},
		{in: " Command ", want: TokenStoreBackendCommand},
	}

	for _, tt := range tests {
		if got := NormalizeTokenStoreBackend(tt.in); got != tt.want {
			t.Fatalf("NormalizeTokenStoreBackend(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGetDefaultAppConfigUsesSystemKeyring(t *testing.T) {
	cfg := getDefaultAppConfig()

	if got := cfg.Auth.TokenStore.Backend; got != TokenStoreBackendKeyring {
		t.Fatalf("Auth.TokenStore.Backend = %q, want %q", got, TokenStoreBackendKeyring)
	}
	if got := cfg.Auth.TokenStore.Command.Timeout; got != defaultTokenStoreCommandTimeout {
		t.Fatalf("Auth.TokenStore.Command.Timeout = %d, want %d", got, defaultTokenStoreCommandTimeout)
	}
}

func TestValidateStartupConfigAcceptsTokenStoreBackends(t *testing.T) {
	fileCfg := validConfig()
	fileCfg.Auth.TokenStore.Backend = TokenStoreBackendFile
	if err := validateStartupConfig(fileCfg); err != nil {
		t.Fatalf("validateStartupConfig(file) error = %v, want nil", err)
	}

	commandCfg := validConfig()
	commandCfg.Auth.TokenStore.Backend = TokenStoreBackendCommand
	commandCfg.Auth.TokenStore.Command.Get = []string{"pass", "show", "lazyspotify/token"}
	commandCfg.Auth.TokenStore.Command.Set = []string{"pass", "insert", "-m", "-f", "lazyspotify/token"}
	if err := validateStartupConfig(commandCfg); err != nil {
		t.Fatalf("validateStartupConfig(command) error = %v, want nil", err)
	}
}

func TestValidateStartupConfigRejectsUnknownTokenStoreBackend(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.TokenStore.Backend = "carrier-pigeon"

	err := validateStartupConfig(cfg)
	if err == nil {
		t.Fatal("validateStartupConfig() = nil, want error")
	}
	if !strings.Contains(err.Error(), "auth.token-store.backend") {
		t.Fatalf("validateStartupConfig() = %q, want auth.token-store.backend context", err.Error())
	}
}

func TestValidateStartupConfigRejectsIncompleteCommandTokenStore(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.TokenStore.Backend = TokenStoreBackendCommand

	err := validateStartupConfig(cfg)
	if err == nil {
		t.Fatal("validateStartupConfig() = nil, want error")
	}
	if !strings.Contains(err.Error(), "auth.token-store.command.get") {
		t.Fatalf("validateStartupConfig() = %q, want missing get command context", err.Error())
	}

	cfg.Auth.TokenStore.Command.Get = []string{"pass", "show", "lazyspotify/token"}
	err = validateStartupConfig(cfg)
	if err == nil {
		t.Fatal("validateStartupConfig() = nil, want error")
	}
	if !strings.Contains(err.Error(), "auth.token-store.command.set") {
		t.Fatalf("validateStartupConfig() = %q, want missing set command context", err.Error())
	}
}

func TestLoadConfigReadsTokenStoreSettings(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	configDir := getConfigDir()
	if err := EnsureExists(configDir); err != nil {
		t.Fatalf("EnsureExists(%q) error = %v, want nil", configDir, err)
	}
	configPath := filepath.Join(configDir, appConfigFileName)
	wantConfig := strings.Join([]string{
		"auth:",
		"  client_id: existing-client-id",
		"  token-store:",
		"    backend: command",
		"    command:",
		"      get:",
		"        - pass",
		"        - show",
		"        - lazyspotify/token",
		"      set:",
		"        - pass",
		"        - insert",
		"        - -m",
		"        - -f",
		"        - lazyspotify/token",
		"      timeout: 25",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(wantConfig), 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", configPath, err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	if got := cfg.Auth.TokenStore.Backend; got != TokenStoreBackendCommand {
		t.Fatalf("Auth.TokenStore.Backend = %q, want %q", got, TokenStoreBackendCommand)
	}
	wantGet := "pass show lazyspotify/token"
	if got := strings.Join(cfg.Auth.TokenStore.Command.Get, " "); got != wantGet {
		t.Fatalf("Auth.TokenStore.Command.Get = %q, want %q", got, wantGet)
	}
	wantSet := "pass insert -m -f lazyspotify/token"
	if got := strings.Join(cfg.Auth.TokenStore.Command.Set, " "); got != wantSet {
		t.Fatalf("Auth.TokenStore.Command.Set = %q, want %q", got, wantSet)
	}
	if got := cfg.Auth.TokenStore.Command.Timeout; got != 25 {
		t.Fatalf("Auth.TokenStore.Command.Timeout = %d, want 25", got)
	}
}

func TestLoadConfigAllowsTokenStoreBackendEnvOverride(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AUTH_TOKEN_STORE_BACKEND", "file")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	if got := cfg.Auth.TokenStore.Backend; got != TokenStoreBackendFile {
		t.Fatalf("Auth.TokenStore.Backend = %q, want %q", got, TokenStoreBackendFile)
	}
}

func TestLoadConfigAllowsTokenStoreFilePathEnvOverride(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	want := filepath.Join(homeDir, "elsewhere", "token.json")
	t.Setenv("AUTH_TOKEN_STORE_FILE_PATH", want)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	if got := cfg.Auth.TokenStore.File.Path; got != want {
		t.Fatalf("Auth.TokenStore.File.Path = %q, want %q", got, want)
	}
}

func TestLoadConfigAllowsTokenStoreCommandEnvOverrides(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AUTH_TOKEN_STORE_COMMAND_TIMEOUT", "42")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	if got := cfg.Auth.TokenStore.Command.Timeout; got != 42 {
		t.Fatalf("Auth.TokenStore.Command.Timeout = %d, want 42", got)
	}
}

func validConfig() AppConfig {
	cfg := getDefaultAppConfig()
	cfg.Auth.ClientID = "configured-client-id"
	return cfg
}
