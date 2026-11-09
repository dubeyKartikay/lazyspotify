package auth

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func requireSh(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is required for command token store tests")
	}
}

func TestCommandTokenStoreRoundTrip(t *testing.T) {
	requireSh(t)
	secretPath := filepath.Join(t.TempDir(), "secret")
	t.Setenv("LAZYSPOTIFY_TEST_SECRET_FILE", secretPath)

	store, err := NewCommandTokenStore(
		[]string{"sh", "-c", `cat "$LAZYSPOTIFY_TEST_SECRET_FILE"`},
		[]string{"sh", "-c", `cat > "$LAZYSPOTIFY_TEST_SECRET_FILE"`},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	if _, err := store.GetString("token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("GetString() with no secret = %v, want ErrTokenNotFound", err)
	}

	if err := store.SetString("token-v2", `{"access_token":"abc"}`); err != nil {
		t.Fatalf("SetString() error = %v, want nil", err)
	}

	got, err := store.GetString("token-v2")
	if err != nil {
		t.Fatalf("GetString() error = %v, want nil", err)
	}
	if got != `{"access_token":"abc"}` {
		t.Fatalf("GetString() = %q, want stored secret", got)
	}
}

func TestCommandTokenStoreTrimsTrailingNewline(t *testing.T) {
	requireSh(t)
	store, err := NewCommandTokenStore(
		[]string{"sh", "-c", `printf 'secret\n'`},
		[]string{"sh", "-c", `cat > /dev/null`},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	got, err := store.GetString("token-v2")
	if err != nil {
		t.Fatalf("GetString() error = %v, want nil", err)
	}
	if got != "secret" {
		t.Fatalf("GetString() = %q, want %q", got, "secret")
	}
}

func TestCommandTokenStorePassesTokenKey(t *testing.T) {
	requireSh(t)
	store, err := NewCommandTokenStore(
		[]string{"sh", "-c", `printf "%s" "$LAZYSPOTIFY_TOKEN_KEY"`},
		[]string{"sh", "-c", `cat > /dev/null`},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	got, err := store.GetString("token-v2")
	if err != nil {
		t.Fatalf("GetString() error = %v, want nil", err)
	}
	if got != "token-v2" {
		t.Fatalf("GetString() = %q, want %q", got, "token-v2")
	}
}

func TestCommandTokenStoreEmptySecretIsMissing(t *testing.T) {
	requireSh(t)
	store, err := NewCommandTokenStore(
		[]string{"sh", "-c", `printf '\n'`},
		[]string{"sh", "-c", `cat > /dev/null`},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	if _, err := store.GetString("token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("GetString() = %v, want ErrTokenNotFound", err)
	}
}

func TestCommandTokenStoreSetReportsStderr(t *testing.T) {
	requireSh(t)
	store, err := NewCommandTokenStore(
		[]string{"sh", "-c", "exit 0"},
		[]string{"sh", "-c", `echo "gpg: decryption failed" >&2; exit 2`},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	err = store.SetString("token-v2", "secret")
	if err == nil {
		t.Fatal("SetString() = nil, want error")
	}
	if !strings.Contains(err.Error(), "gpg: decryption failed") {
		t.Fatalf("SetString() = %q, want command stderr in error", err.Error())
	}
}

func TestCommandTokenStoreMissingCommandIsAnError(t *testing.T) {
	store, err := NewCommandTokenStore([]string{"lazyspotify-no-such-secret-tool"}, []string{"lazyspotify-no-such-secret-tool"}, time.Second)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	_, err = store.GetString("token-v2")
	if err == nil {
		t.Fatal("GetString() = nil, want error for missing binary")
	}
	if errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("GetString() = %v, want command error rather than ErrTokenNotFound", err)
	}
}

func TestCommandTokenStoreTimeout(t *testing.T) {
	requireSh(t)
	store, err := NewCommandTokenStore(
		[]string{"sh", "-c", "sleep 5"},
		[]string{"sh", "-c", `cat > /dev/null`},
		50*time.Millisecond,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	start := time.Now()
	_, err = store.GetString("token-v2")
	if err == nil {
		t.Fatal("GetString() = nil, want timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("GetString() = %q, want timeout error", err.Error())
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("GetString() took %s, want it to stop near the timeout", elapsed)
	}
}

func TestNewCommandTokenStoreRequiresBothCommands(t *testing.T) {
	if _, err := NewCommandTokenStore(nil, []string{"sh"}, time.Second); err == nil {
		t.Fatal("NewCommandTokenStore(nil get) = nil error, want error")
	}
	if _, err := NewCommandTokenStore([]string{"sh"}, nil, time.Second); err == nil {
		t.Fatal("NewCommandTokenStore(nil set) = nil error, want error")
	}
}

func TestNewCommandTokenStoreDefaultsTimeout(t *testing.T) {
	store, err := NewCommandTokenStore([]string{"sh"}, []string{"sh"}, 0)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}
	if store.timeout != defaultCommandTimeout {
		t.Fatalf("timeout = %s, want %s", store.timeout, defaultCommandTimeout)
	}
}
