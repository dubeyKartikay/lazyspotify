package auth

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"golang.org/x/oauth2"
)

func TestNewTokenStoreFromConfigSelectsKeyringByDefault(t *testing.T) {
	store, err := NewTokenStoreFromConfig(utils.AppConfig{})
	if err != nil {
		t.Fatalf("NewTokenStoreFromConfig() error = %v, want nil", err)
	}
	if _, ok := store.(*KeyringStore); !ok {
		t.Fatalf("NewTokenStoreFromConfig() = %T, want *KeyringStore", store)
	}
}

func TestNewTokenStoreFromConfigSelectsFileStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	cfg := utils.AppConfig{}
	cfg.Auth.TokenStore.Backend = utils.TokenStoreBackendFile
	cfg.Auth.TokenStore.File.Path = path

	store, err := NewTokenStoreFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewTokenStoreFromConfig() error = %v, want nil", err)
	}
	fileStore, ok := store.(*FileTokenStore)
	if !ok {
		t.Fatalf("NewTokenStoreFromConfig() = %T, want *FileTokenStore", store)
	}
	if fileStore.path != path {
		t.Fatalf("path = %q, want %q", fileStore.path, path)
	}
}

func TestNewTokenStoreFromConfigSelectsCommandStore(t *testing.T) {
	cfg := utils.AppConfig{}
	cfg.Auth.TokenStore.Backend = utils.TokenStoreBackendCommand
	cfg.Auth.TokenStore.Command.Get = []string{"pass", "show", "lazyspotify/token"}
	cfg.Auth.TokenStore.Command.Set = []string{"pass", "insert", "-m", "-f", "lazyspotify/token"}
	cfg.Auth.TokenStore.Command.NotFoundExitCodes = []int{1}

	store, err := NewTokenStoreFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewTokenStoreFromConfig() error = %v, want nil", err)
	}
	commandStore, ok := store.(*CommandTokenStore)
	if !ok {
		t.Fatalf("NewTokenStoreFromConfig() = %T, want *CommandTokenStore", store)
	}
	if _, ok := commandStore.notFoundExitCodes[1]; !ok {
		t.Fatalf("notFoundExitCodes = %v, want 1", commandStore.notFoundExitCodes)
	}
	if commandStore.anyExitIsNotFound {
		t.Fatal("anyExitIsNotFound = true, want false when codes are configured")
	}
}

func TestNewTokenStoreFromConfigRejectsUnknownBackend(t *testing.T) {
	cfg := utils.AppConfig{}
	cfg.Auth.TokenStore.Backend = "carrier-pigeon"

	_, err := NewTokenStoreFromConfig(cfg)
	if err == nil {
		t.Fatal("NewTokenStoreFromConfig() = nil error, want unsupported backend error")
	}
	if got := err.Error(); !strings.Contains(got, "keyring, file, command") {
		t.Fatalf("NewTokenStoreFromConfig() = %q, want the supported backend list", got)
	}
}

func TestNewTokenStoreFromConfigRejectsIncompleteCommandStore(t *testing.T) {
	cfg := utils.AppConfig{}
	cfg.Auth.TokenStore.Backend = utils.TokenStoreBackendCommand

	if _, err := NewTokenStoreFromConfig(cfg); err == nil {
		t.Fatal("NewTokenStoreFromConfig() = nil error, want missing command error")
	}
}

func TestOAuthTokenRoundTripsThroughTokenStore(t *testing.T) {
	ctx := context.Background()
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "token.json"))
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}

	if _, err := store.Get(ctx, "token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("Get() on empty store = %v, want ErrTokenNotFound", err)
	}

	token := &oauth2.Token{
		AccessToken:  "access",
		TokenType:    "Bearer",
		RefreshToken: "refresh",
		Expiry:       time.Unix(1700000000, 0).UTC(),
	}
	if err := store.Set(ctx, "token-v2", token); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	got, err := store.Get(ctx, "token-v2")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got.AccessToken != token.AccessToken || got.RefreshToken != token.RefreshToken {
		t.Fatalf("Get() = %+v, want %+v", got, token)
	}
	if !got.Expiry.Equal(token.Expiry) {
		t.Fatalf("Get().Expiry = %s, want %s", got.Expiry, token.Expiry)
	}
}

func TestDecodeTokenRejectsBlankTokens(t *testing.T) {
	tests := []struct {
		name       string
		serialized string
	}{
		{name: "null", serialized: "null"},
		{name: "empty object", serialized: "{}"},
		{name: "empty access token", serialized: `{"access_token":""}`},
		{name: "unrelated field", serialized: `{"foo":1}`},
		{name: "not json", serialized: "not a token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := decodeToken("token-v2", tt.serialized); !errors.Is(err, ErrTokenNotFound) {
				t.Fatalf("decodeToken(%q) = %v, want ErrTokenNotFound", tt.serialized, err)
			}
		})
	}
}

func TestDecodeTokenAcceptsUsableTokens(t *testing.T) {
	tests := []struct {
		name       string
		serialized string
	}{
		{name: "access", serialized: `{"access_token":"abc"}`},
		{name: "refresh only", serialized: `{"refresh_token":"refresh"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := decodeToken("token-v2", tt.serialized); err != nil {
				t.Fatalf("decodeToken(%q) = %v, want nil", tt.serialized, err)
			}
		})
	}
}

func TestEncodeTokenRejectsUnusableTokens(t *testing.T) {
	if _, err := encodeToken(nil); err == nil {
		t.Fatal("encodeToken(nil) = nil error, want error")
	}
	if _, err := encodeToken(&oauth2.Token{}); err == nil {
		t.Fatal("encodeToken(empty) = nil error, want error")
	}
}

func TestErrorTokenStoreReturnsConstructionError(t *testing.T) {
	ctx := context.Background()
	root := errors.New("bad token store config")
	store := errorTokenStore{err: root}

	if _, err := store.Get(ctx, "token-v2"); !errors.Is(err, root) {
		t.Fatalf("Get() = %v, want the construction error", err)
	}
	if err := store.Set(ctx, "token-v2", &oauth2.Token{AccessToken: "x"}); !errors.Is(err, root) {
		t.Fatalf("Set() = %v, want the construction error", err)
	}
}

func TestStoreRejectsBlankTokenOnSet(t *testing.T) {
	ctx := context.Background()
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "token.json"))
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}
	if err := store.Set(ctx, "token-v2", &oauth2.Token{}); err == nil {
		t.Fatal("Set(empty token) = nil error, want error")
	}
}
