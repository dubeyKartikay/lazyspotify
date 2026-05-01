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
