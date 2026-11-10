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
	cfg.Auth.TokenStore.Key = "token-v2"

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
		t.Run(tt.in, func(t *testing.T) {
			if got := NormalizeTokenStoreBackend(tt.in); got != tt.want {
				t.Fatalf("NormalizeTokenStoreBackend(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSupportedTokenStoreBackendsIsACopy(t *testing.T) {
	backends := SupportedTokenStoreBackends()
	backends[0] = "mutated"

	if got := SupportedTokenStoreBackends()[0]; got != TokenStoreBackendKeyring {
		t.Fatalf("SupportedTokenStoreBackends()[0] = %q, want %q after mutating a previous copy", got, TokenStoreBackendKeyring)
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
	if got := cfg.Auth.TokenStore.Key; got != "token-v2" {
		t.Fatalf("Auth.TokenStore.Key = %q, want %q", got, "token-v2")
	}
	if got := cfg.Auth.Keyring.Key; got != "" {
		t.Fatalf("Auth.Keyring.Key = %q, want empty (deprecated alias has no default)", got)
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

func TestLoadConfigSplitsCommandEnvIntoArgv(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want []string
	}{
		{
			name: "space separated",
			env:  "pass show lazyspotify/token",
			want: []string{"pass", "show", "lazyspotify/token"},
		},
		{
			name: "quoted arguments keep spaces",
			env:  `pass insert -m -f "lazyspotify/work token"`,
			want: []string{"pass", "insert", "-m", "-f", "lazyspotify/work token"},
		},
		{
			name: "single quotes",
			env:  `cmd 'one two' three`,
			want: []string{"cmd", "one two", "three"},
		},
		{
			name: "escaped space",
			env:  `cmd one\ two three`,
			want: []string{"cmd", "one two", "three"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homeDir := t.TempDir()
			t.Setenv("HOME", homeDir)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
			t.Setenv("AUTH_TOKEN_STORE_BACKEND", "command")
			t.Setenv("AUTH_TOKEN_STORE_COMMAND_GET", tt.env)
			t.Setenv("AUTH_TOKEN_STORE_COMMAND_SET", `sh -c "cat > /dev/null"`)

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig() error = %v, want nil", err)
			}
			if got := strings.Join(cfg.Auth.TokenStore.Command.Get, "\x00"); got != strings.Join(tt.want, "\x00") {
				t.Fatalf("Command.Get = %#v, want %#v", cfg.Auth.TokenStore.Command.Get, tt.want)
			}
		})
	}
}

func TestLoadConfigSplitsCommandStringFromYAML(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	configDir := getConfigDir()
	if err := EnsureExists(configDir); err != nil {
		t.Fatalf("EnsureExists(%q) error = %v, want nil", configDir, err)
	}
	configPath := filepath.Join(configDir, appConfigFileName)
	config := strings.Join([]string{
		"auth:",
		"  client_id: existing-client-id",
		"  token-store:",
		"    backend: command",
		"    command:",
		"      get: pass show lazyspotify/token",
		"      set: pass insert -m -f lazyspotify/token",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", configPath, err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	want := "pass show lazyspotify/token"
	if got := strings.Join(cfg.Auth.TokenStore.Command.Get, " "); got != want {
		t.Fatalf("Command.Get = %q, want %q split into argv", got, want)
	}
}

func TestLoadConfigRejectsUnbalancedCommandEnvQuotes(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AUTH_TOKEN_STORE_BACKEND", "command")
	t.Setenv("AUTH_TOKEN_STORE_COMMAND_GET", `pass "show`)
	t.Setenv("AUTH_TOKEN_STORE_COMMAND_SET", "pass insert")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig() = nil error, want unbalanced quote error")
	}
	if !strings.Contains(err.Error(), "unbalanced") {
		t.Fatalf("LoadConfig() = %q, want quote context", err.Error())
	}
}

func TestValidateTokenStoreConfigRejectsBadFilePaths(t *testing.T) {
	for _, path := range []string{"~", "~/", "~other/token.json", "/var/lib/lazyspotify/"} {
		cfg := validConfig()
		cfg.Auth.TokenStore.Backend = TokenStoreBackendFile
		cfg.Auth.TokenStore.File.Path = path

		err := validateStartupConfig(cfg)
		if err == nil {
			t.Fatalf("validateStartupConfig(path=%q) = nil error, want error", path)
		}
		if !strings.Contains(err.Error(), "auth.token-store.file.path") {
			t.Fatalf("validateStartupConfig(path=%q) = %q, want the config key", path, err.Error())
		}
	}
}

func TestValidateTokenStoreConfigRejectsNonPositiveNotFoundExitCode(t *testing.T) {
	for _, code := range []int{-1, 0} {
		cfg := validConfig()
		cfg.Auth.TokenStore.Backend = TokenStoreBackendCommand
		cfg.Auth.TokenStore.Command.Get = []string{"pass", "show", "lazyspotify/token"}
		cfg.Auth.TokenStore.Command.Set = []string{"pass", "insert", "-m", "-f", "lazyspotify/token"}
		cfg.Auth.TokenStore.Command.NotFoundExitCodes = []int{code}

		err := validateStartupConfig(cfg)
		if err == nil {
			t.Fatalf("validateStartupConfig(code=%d) = nil error, want error", code)
		}
		if !strings.Contains(err.Error(), "not-found-exit-codes") || !strings.Contains(err.Error(), "positive") {
			t.Fatalf("validateStartupConfig(code=%d) = %q, want the config key and positive-code guidance", code, err.Error())
		}
	}
}

func TestValidateStartupConfigRequiresTokenKey(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.TokenStore.Key = ""
	cfg.Auth.Keyring.Key = ""

	err := validateStartupConfig(cfg)
	if err == nil {
		t.Fatal("validateStartupConfig() = nil error, want missing token key error")
	}
	if !strings.Contains(err.Error(), "auth.token-store.key") {
		t.Fatalf("validateStartupConfig() = %q, want the config key", err.Error())
	}
}

func TestLoadConfigReadsTokenStoreKeyAndExitCodes(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	configDir := getConfigDir()
	if err := EnsureExists(configDir); err != nil {
		t.Fatalf("EnsureExists(%q) error = %v, want nil", configDir, err)
	}
	configPath := filepath.Join(configDir, appConfigFileName)
	config := strings.Join([]string{
		"auth:",
		"  client_id: existing-client-id",
		"  token-store:",
		"    backend: command",
		"    key: custom-key",
		"    command:",
		"      get: [pass, show, lazyspotify/token]",
		"      set: [pass, insert, -m, -f, lazyspotify/token]",
		"      not-found-exit-codes: [1, 3]",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", configPath, err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	if got := cfg.AuthTokenKey(); got != "custom-key" {
		t.Fatalf("AuthTokenKey() = %q, want %q", got, "custom-key")
	}
	if got := cfg.Auth.TokenStore.Command.NotFoundExitCodes; len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("NotFoundExitCodes = %v, want [1 3]", got)
	}
}

func TestAuthTokenKeyFallsBackToDeprecatedKeyringKey(t *testing.T) {
	cfg := AppConfig{}
	cfg.Auth.Keyring.Key = "token-v2"
	if got := cfg.AuthTokenKey(); got != "token-v2" {
		t.Fatalf("AuthTokenKey() = %q, want %q from auth.keyring.key", got, "token-v2")
	}

	cfg.Auth.TokenStore.Key = "token-v3"
	if got := cfg.AuthTokenKey(); got != "token-v3" {
		t.Fatalf("AuthTokenKey() = %q, want %q to prefer auth.token-store.key", got, "token-v3")
	}
}

func TestLoadConfigAllowsNotFoundExitCodesEnvOverride(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AUTH_TOKEN_STORE_COMMAND_NOT_FOUND_EXIT_CODES", "1,3")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	got := cfg.Auth.TokenStore.Command.NotFoundExitCodes
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("NotFoundExitCodes = %v, want [1 3]", got)
	}
}

func TestLoadConfigAppliesDeprecatedKeyringKeyAlias(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	configDir := getConfigDir()
	if err := EnsureExists(configDir); err != nil {
		t.Fatalf("EnsureExists(%q) error = %v, want nil", configDir, err)
	}
	config := strings.Join([]string{
		"auth:",
		"  client_id: existing-client-id",
		"  keyring:",
		"    key: legacy-key",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(configDir, appConfigFileName), []byte(config), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v, want nil", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	if got := cfg.AuthTokenKey(); got != "legacy-key" {
		t.Fatalf("AuthTokenKey() = %q, want %q from the deprecated alias", got, "legacy-key")
	}
}

func TestLoadConfigCanonicalTokenKeyWinsOverAlias(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	configDir := getConfigDir()
	if err := EnsureExists(configDir); err != nil {
		t.Fatalf("EnsureExists(%q) error = %v, want nil", configDir, err)
	}
	config := strings.Join([]string{
		"auth:",
		"  client_id: existing-client-id",
		"  keyring:",
		"    key: legacy-key",
		"  token-store:",
		"    key: canonical-key",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(configDir, appConfigFileName), []byte(config), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v, want nil", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	if got := cfg.AuthTokenKey(); got != "canonical-key" {
		t.Fatalf("AuthTokenKey() = %q, want %q", got, "canonical-key")
	}
}

func TestLoadConfigAppliesDeprecatedKeyringKeyEnvAlias(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AUTH_KEYRING_KEY", "legacy-env-key")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	if got := cfg.AuthTokenKey(); got != "legacy-env-key" {
		t.Fatalf("AuthTokenKey() = %q, want %q from AUTH_KEYRING_KEY", got, "legacy-env-key")
	}
}

func TestLoadConfigReportsIntegerParseErrors(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AUTH_TOKEN_STORE_COMMAND_NOT_FOUND_EXIT_CODES", "one")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig() = nil error, want integer parse error")
	}
	if !strings.Contains(err.Error(), "failed to parse config") || !strings.Contains(err.Error(), "invalid integer") {
		t.Fatalf("LoadConfig() = %q, want parse context", err.Error())
	}
}

func TestLoadConfigAllowsTokenStoreKeyEnvOverride(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	t.Setenv("AUTH_TOKEN_STORE_KEY", "env-key")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v, want nil", err)
	}
	if got := cfg.AuthTokenKey(); got != "env-key" {
		t.Fatalf("AuthTokenKey() = %q, want %q", got, "env-key")
	}
}

func validConfig() AppConfig {
	cfg := getDefaultAppConfig()
	cfg.Auth.ClientID = "configured-client-id"
	return cfg
}
