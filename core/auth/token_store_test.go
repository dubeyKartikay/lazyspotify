package auth

import (
	"errors"
	"path/filepath"
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
	if _, ok := store.(*Keyring); !ok {
		t.Fatalf("NewTokenStoreFromConfig() = %T, want *Keyring", store)
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
	if fileStore.Path() != path {
		t.Fatalf("Path() = %q, want %q", fileStore.Path(), path)
	}
}

func TestNewTokenStoreFromConfigSelectsCommandStore(t *testing.T) {
	cfg := utils.AppConfig{}
	cfg.Auth.TokenStore.Backend = utils.TokenStoreBackendCommand
	cfg.Auth.TokenStore.Command.Get = []string{"pass", "show", "lazyspotify/token"}
	cfg.Auth.TokenStore.Command.Set = []string{"pass", "insert", "-m", "-f", "lazyspotify/token"}

	store, err := NewTokenStoreFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewTokenStoreFromConfig() error = %v, want nil", err)
	}
	if _, ok := store.(*CommandTokenStore); !ok {
		t.Fatalf("NewTokenStoreFromConfig() = %T, want *CommandTokenStore", store)
	}
}

func TestNewTokenStoreFromConfigRejectsUnknownBackend(t *testing.T) {
	cfg := utils.AppConfig{}
	cfg.Auth.TokenStore.Backend = "carrier-pigeon"

	if _, err := NewTokenStoreFromConfig(cfg); err == nil {
		t.Fatal("NewTokenStoreFromConfig() = nil error, want unsupported backend error")
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
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "token.json"))
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}

	if _, err := GetOAuthToken(store, "token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("GetOAuthToken() on empty store = %v, want ErrTokenNotFound", err)
	}

	token := &oauth2.Token{
		AccessToken:  "access",
		TokenType:    "Bearer",
		RefreshToken: "refresh",
		Expiry:       time.Unix(1700000000, 0).UTC(),
	}
	if err := SetOAuthToken(store, "token-v2", token); err != nil {
		t.Fatalf("SetOAuthToken() error = %v, want nil", err)
	}

	got, err := GetOAuthToken(store, "token-v2")
	if err != nil {
		t.Fatalf("GetOAuthToken() error = %v, want nil", err)
	}
	if got.AccessToken != token.AccessToken || got.RefreshToken != token.RefreshToken {
		t.Fatalf("GetOAuthToken() = %+v, want %+v", got, token)
	}
	if !got.Expiry.Equal(token.Expiry) {
		t.Fatalf("GetOAuthToken().Expiry = %s, want %s", got.Expiry, token.Expiry)
	}
}

func TestGetOAuthTokenTreatsUndecodableTokenAsMissing(t *testing.T) {
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "token.json"))
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}
	if err := store.SetString("token-v2", "not a token"); err != nil {
		t.Fatalf("SetString() error = %v, want nil", err)
	}

	if _, err := GetOAuthToken(store, "token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("GetOAuthToken() = %v, want ErrTokenNotFound so login can restart", err)
	}

	token := &oauth2.Token{AccessToken: "fresh"}
	if err := SetOAuthToken(store, "token-v2", token); err != nil {
		t.Fatalf("SetOAuthToken() error = %v, want nil", err)
	}
	got, err := GetOAuthToken(store, "token-v2")
	if err != nil {
		t.Fatalf("GetOAuthToken() after re-login error = %v, want nil", err)
	}
	if got.AccessToken != "fresh" {
		t.Fatalf("GetOAuthToken().AccessToken = %q, want %q", got.AccessToken, "fresh")
	}
}
