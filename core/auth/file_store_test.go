package auth

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestFileTokenStoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "nested", "token.json")
	store, err := NewFileTokenStore(path)
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}

	if _, err := store.Get(ctx, "token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("Get() on empty store = %v, want ErrTokenNotFound", err)
	}

	want := &oauth2.Token{AccessToken: "abc", RefreshToken: "refresh"}
	if err := store.Set(ctx, "token-v2", want); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	got, err := store.Get(ctx, "token-v2")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got.AccessToken != want.AccessToken || got.RefreshToken != want.RefreshToken {
		t.Fatalf("Get() = %+v, want %+v", got, want)
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
	ctx := context.Background()
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "token.json"))
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}

	if err := store.Set(ctx, "token-v2", &oauth2.Token{AccessToken: "first"}); err != nil {
		t.Fatalf("Set(token-v2) error = %v, want nil", err)
	}
	if err := store.Set(ctx, "other", &oauth2.Token{AccessToken: "second"}); err != nil {
		t.Fatalf("Set(other) error = %v, want nil", err)
	}

	first, err := store.Get(ctx, "token-v2")
	if err != nil {
		t.Fatalf("Get(token-v2) error = %v, want nil", err)
	}
	if first.AccessToken != "first" {
		t.Fatalf("Get(token-v2) = %q, want %q", first.AccessToken, "first")
	}

	other, err := store.Get(ctx, "other")
	if err != nil {
		t.Fatalf("Get(other) error = %v, want nil", err)
	}
	if other.AccessToken != "second" {
		t.Fatalf("Get(other) = %q, want %q", other.AccessToken, "second")
	}
}

func TestFileTokenStoreMissingKey(t *testing.T) {
	ctx := context.Background()
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "token.json"))
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}
	if err := store.Set(ctx, "token-v2", &oauth2.Token{AccessToken: "value"}); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	if _, err := store.Get(ctx, "another-key"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("Get(another-key) = %v, want ErrTokenNotFound", err)
	}
}

func TestFileTokenStoreRejectsEmptyToken(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "token.json")
	store, err := NewFileTokenStore(path)
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}

	if err := store.Set(ctx, "token-v2", &oauth2.Token{}); err == nil {
		t.Fatal("Set(empty token) = nil error, want error")
	}
	if err := store.Set(ctx, "token-v2", nil); err == nil {
		t.Fatal("Set(nil) = nil error, want error")
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("os.Stat(%q) = %v, want no file to be written", path, err)
	}
}

func TestFileTokenStoreQuarantinesCorruptFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "token.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}
	store, err := NewFileTokenStore(path)
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}

	if _, err := store.Get(ctx, "token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("Get() on corrupt file = %v, want ErrTokenNotFound so login can restart", err)
	}

	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("os.Stat(%q) = %v, want the corrupt file to be moved aside", path, err)
	}
	quarantined, err := filepath.Glob(path + ".corrupt-*")
	if err != nil {
		t.Fatalf("filepath.Glob() error = %v, want nil", err)
	}
	if len(quarantined) != 1 {
		t.Fatalf("quarantined files = %v, want exactly one", quarantined)
	}
	contents, err := os.ReadFile(quarantined[0])
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", quarantined[0], err)
	}
	if string(contents) != "not json" {
		t.Fatalf("quarantined contents = %q, want the original bytes", string(contents))
	}

	// A fresh login must be able to write the token again.
	if err := store.Set(ctx, "token-v2", &oauth2.Token{AccessToken: "fresh"}); err != nil {
		t.Fatalf("Set() after quarantine error = %v, want nil", err)
	}
	got, err := store.Get(ctx, "token-v2")
	if err != nil {
		t.Fatalf("Get() after quarantine error = %v, want nil", err)
	}
	if got.AccessToken != "fresh" {
		t.Fatalf("Get() = %q, want %q", got.AccessToken, "fresh")
	}
}

func TestFileTokenStoreQuarantineDoesNotClobberExistingBackup(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "token.json")
	store, err := NewFileTokenStore(path)
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}
	if err := os.WriteFile(path, []byte("first corrupt"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}
	if _, err := store.Get(ctx, "token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("first Get() = %v, want ErrTokenNotFound", err)
	}

	if err := os.WriteFile(path, []byte("second corrupt"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", path, err)
	}
	if _, err := store.Get(ctx, "token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("second Get() = %v, want ErrTokenNotFound", err)
	}

	quarantined, err := filepath.Glob(path + ".corrupt-*")
	if err != nil {
		t.Fatalf("filepath.Glob() error = %v, want nil", err)
	}
	if len(quarantined) != 2 {
		t.Fatalf("quarantined files = %v, want two distinct backups", quarantined)
	}

	seen := map[string]bool{}
	for _, backup := range quarantined {
		contents, err := os.ReadFile(backup)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v, want nil", backup, err)
		}
		seen[string(contents)] = true
	}
	if !seen["first corrupt"] || !seen["second corrupt"] {
		t.Fatalf("quarantined contents = %v, want both backups to keep their bytes", seen)
	}
}

func TestFileTokenStoreSweepsStaleTemps(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	store, err := NewFileTokenStore(path)
	if err != nil {
		t.Fatalf("NewFileTokenStore() error = %v, want nil", err)
	}

	stale := filepath.Join(dir, ".token-stale.tmp")
	if err := os.WriteFile(stale, []byte("leftover"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", stale, err)
	}
	old := time.Now().Add(-2 * staleTokenTempAge)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatalf("os.Chtimes(%q) error = %v, want nil", stale, err)
	}
	fresh := filepath.Join(dir, ".token-fresh.tmp")
	if err := os.WriteFile(fresh, []byte("in flight"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v, want nil", fresh, err)
	}

	if err := store.Set(ctx, "token-v2", &oauth2.Token{AccessToken: "value"}); err != nil {
		t.Fatalf("Set() error = %v, want nil", err)
	}

	if _, err := os.Stat(stale); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("os.Stat(%q) = %v, want stale temp removed", stale, err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatalf("os.Stat(%q) = %v, want recent temp kept", fresh, err)
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
	if got := store.path; got != want {
		t.Fatalf("path = %q, want %q", got, want)
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
	if got := store.path; got != want {
		t.Fatalf("path = %q, want %q", got, want)
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
	if got := store.path; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

func TestNewFileTokenStoreRejectsBareTilde(t *testing.T) {
	for _, path := range []string{"~", "~/"} {
		if _, err := NewFileTokenStore(path); err == nil {
			t.Fatalf("NewFileTokenStore(%q) = nil error, want a file path error", path)
		}
	}
}

func TestHasLoosePermissions(t *testing.T) {
	tests := []struct {
		mode fs.FileMode
		want bool
	}{
		{mode: 0o600, want: false},
		{mode: 0o400, want: false},
		{mode: 0o640, want: true},
		{mode: 0o604, want: true},
		{mode: 0o644, want: true},
	}

	for _, tt := range tests {
		if got := hasLoosePermissions(tt.mode); got != tt.want {
			t.Fatalf("hasLoosePermissions(%o) = %t, want %t", tt.mode, got, tt.want)
		}
	}
}
