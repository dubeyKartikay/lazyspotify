package auth

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileTokenStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "token.json")
	store, err := NewFileTokenStore(path)
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}

	if _, err := store.GetString("token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("GetString() on empty store = %v, want ErrTokenNotFound", err)
	}

	if err := store.SetString("token-v2", `{"access_token":"abc"}`); err != nil {
		t.Fatalf("SetString() error = %v, want nil", err)
	}

	got, err := store.GetString("token-v2")
	if err != nil {
		t.Fatalf("GetString() error = %v, want nil", err)
	}
	if got != `{"access_token":"abc"}` {
		t.Fatalf("GetString() = %q, want stored token", got)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want nil", path, err)
	}
	if perms := info.Mode().Perm(); perms != 0o600 {
		t.Fatalf("token file permissions = %o, want 600", perms)
	}
}

func TestFileTokenStoreKeepsOtherKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	store, err := NewFileTokenStore(path)
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}

	if err := store.SetString("token-v2", "first"); err != nil {
		t.Fatalf("SetString(token-v2) error = %v, want nil", err)
	}
	if err := store.SetString("other", "second"); err != nil {
		t.Fatalf("SetString(other) error = %v, want nil", err)
	}

	first, err := store.GetString("token-v2")
	if err != nil {
		t.Fatalf("GetString(token-v2) error = %v, want nil", err)
	}
	if first != "first" {
		t.Fatalf("GetString(token-v2) = %q, want %q", first, "first")
	}

	other, err := store.GetString("other")
	if err != nil {
		t.Fatalf("GetString(other) error = %v, want nil", err)
	}
	if other != "second" {
		t.Fatalf("GetString(other) = %q, want %q", other, "second")
	}
}

func TestFileTokenStoreMissingKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	store, err := NewFileTokenStore(path)
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}
	if err := store.SetString("token-v2", "value"); err != nil {
		t.Fatalf("SetString() error = %v, want nil", err)
	}

	if _, err := store.GetString("another-key"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("GetString(another-key) = %v, want ErrTokenNotFound", err)
	}
}

func TestFileTokenStoreCorruptFileIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}
	store, err := NewFileTokenStore(path)
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}

	_, err = store.GetString("token-v2")
	if err == nil {
		t.Fatal("GetString() on corrupt file = nil, want error")
	}
	if errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("GetString() on corrupt file = %v, want parse error", err)
	}
}

func TestFileTokenStoreDefaultsToConfigDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("os.UserConfigDir() error = %v, want nil", err)
	}

	store, err := NewFileTokenStore("")
	if err != nil {
		t.Fatalf("NewFileTokenStore(\"\") error = %v, want nil", err)
	}
	want := filepath.Join(configDir, "lazyspotify", defaultTokenFileName)
	if got := store.Path(); got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}

func TestFileTokenStoreResolvesRelativePathFromConfigDir(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("os.UserConfigDir() error = %v, want nil", err)
	}

	store, err := NewFileTokenStore("secrets/token.json")
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}
	want := filepath.Join(configDir, "lazyspotify", "secrets", "token.json")
	if got := store.Path(); got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}

func TestFileTokenStoreExpandsHome(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	store, err := NewFileTokenStore("~/lazyspotify-token.json")
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}
	want := filepath.Join(homeDir, "lazyspotify-token.json")
	if got := store.Path(); got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}
