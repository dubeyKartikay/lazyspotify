package auth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"golang.org/x/oauth2"
)

func requireSh(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh is required for command token store tests")
	}
}

func TestCommandTokenStoreRoundTrip(t *testing.T) {
	requireSh(t)
	ctx := context.Background()
	secretPath := filepath.Join(t.TempDir(), "secret")
	t.Setenv("LAZYSPOTIFY_TEST_SECRET_FILE", secretPath)

	store, err := NewCommandTokenStore(
		[]string{"sh", "-c", `cat "$LAZYSPOTIFY_TEST_SECRET_FILE"`},
		[]string{"sh", "-c", `cat > "$LAZYSPOTIFY_TEST_SECRET_FILE"`},
		nil,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	if _, err := store.Get(ctx, "token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("Get() with no secret = %v, want ErrTokenNotFound", err)
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

	data, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", secretPath, err)
	}
	stored := oauth2.Token{}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("stored secret is not token JSON: %v", err)
	}
	if stored.AccessToken != want.AccessToken {
		t.Fatalf("stored access token = %q, want %q", stored.AccessToken, want.AccessToken)
	}
}

func TestCommandTokenStoreTrimsTrailingNewline(t *testing.T) {
	requireSh(t)
	store, err := NewCommandTokenStore(
		[]string{"sh", "-c", `printf '{"access_token":"secret"}\n'`},
		[]string{"sh", "-c", `cat > /dev/null`},
		nil,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	got, err := store.Get(context.Background(), "token-v2")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got.AccessToken != "secret" {
		t.Fatalf("Get().AccessToken = %q, want %q", got.AccessToken, "secret")
	}
}

func TestCommandTokenStorePassesTokenKey(t *testing.T) {
	requireSh(t)
	store, err := NewCommandTokenStore(
		[]string{"sh", "-c", `printf '{"access_token":"%s"}' "$LAZYSPOTIFY_TOKEN_KEY"`},
		[]string{"sh", "-c", `cat > /dev/null`},
		nil,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	got, err := store.Get(context.Background(), "token-v2")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if got.AccessToken != "token-v2" {
		t.Fatalf("Get().AccessToken = %q, want %q", got.AccessToken, "token-v2")
	}
}

func TestCommandTokenStoreEmptySecretIsMissing(t *testing.T) {
	requireSh(t)
	store, err := NewCommandTokenStore(
		[]string{"sh", "-c", `printf '\n'`},
		[]string{"sh", "-c", `cat > /dev/null`},
		nil,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	if _, err := store.Get(context.Background(), "token-v2"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("Get() = %v, want ErrTokenNotFound", err)
	}
}

func TestCommandTokenStoreExitCodes(t *testing.T) {
	requireSh(t)
	ctx := context.Background()
	set := []string{"sh", "-c", `cat > /dev/null`}

	t.Run("any non-zero by default", func(t *testing.T) {
		store, err := NewCommandTokenStore(
			[]string{"sh", "-c", `echo "boom" >&2; exit 7`},
			set,
			nil,
			time.Second,
		)
		if err != nil {
			t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
		}
		if _, err := store.Get(ctx, "token-v2"); !errors.Is(err, ErrTokenNotFound) {
			t.Fatalf("Get() = %v, want ErrTokenNotFound for the default any-exit mapping", err)
		}
	})

	t.Run("listed code is missing", func(t *testing.T) {
		store, err := NewCommandTokenStore(
			[]string{"sh", "-c", `echo "not in the password store" >&2; exit 1`},
			set,
			[]int{1},
			time.Second,
		)
		if err != nil {
			t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
		}
		if _, err := store.Get(ctx, "token-v2"); !errors.Is(err, ErrTokenNotFound) {
			t.Fatalf("Get() = %v, want ErrTokenNotFound for exit code 1", err)
		}
	})

	t.Run("unlisted code is a real failure", func(t *testing.T) {
		store, err := NewCommandTokenStore(
			[]string{"sh", "-c", `echo "gpg: decryption failed" >&2; exit 2`},
			set,
			[]int{1},
			time.Second,
		)
		if err != nil {
			t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
		}
		_, err = store.Get(ctx, "token-v2")
		if err == nil {
			t.Fatal("Get() = nil error, want a command failure")
		}
		if errors.Is(err, ErrTokenNotFound) {
			t.Fatalf("Get() = %v, want a real error rather than ErrTokenNotFound", err)
		}
		if !strings.Contains(err.Error(), "token get command") || !strings.Contains(err.Error(), "gpg: decryption failed") {
			t.Fatalf("Get() = %q, want the command and stderr in the error", err.Error())
		}
	})
}

func TestNewCommandTokenStoreRejectsNegativeExitCode(t *testing.T) {
	_, err := NewCommandTokenStore([]string{"sh"}, []string{"sh"}, []int{-1}, time.Second)
	if err == nil {
		t.Fatal("NewCommandTokenStore(negative code) = nil error, want error")
	}
	if !strings.Contains(err.Error(), "not-found-exit-codes") {
		t.Fatalf("NewCommandTokenStore() = %q, want the config key in the error", err.Error())
	}
}

func TestCommandTokenStoreSetReportsStderr(t *testing.T) {
	requireSh(t)
	store, err := NewCommandTokenStore(
		[]string{"sh", "-c", "exit 0"},
		[]string{"sh", "-c", `echo "gpg: decryption failed" >&2; exit 2`},
		nil,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	err = store.Set(context.Background(), "token-v2", &oauth2.Token{AccessToken: "secret"})
	if err == nil {
		t.Fatal("Set() = nil, want error")
	}
	if !strings.Contains(err.Error(), "gpg: decryption failed") {
		t.Fatalf("Set() = %q, want command stderr in error", err.Error())
	}
}

func TestCommandTokenStoreMissingCommandIsAnError(t *testing.T) {
	store, err := NewCommandTokenStore(
		[]string{"lazyspotify-no-such-secret-tool"},
		[]string{"lazyspotify-no-such-secret-tool"},
		nil,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}

	_, err = store.Get(context.Background(), "token-v2")
	if err == nil {
		t.Fatal("Get() = nil, want error for missing binary")
	}
	if errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("Get() = %v, want command error rather than ErrTokenNotFound", err)
	}
}

func TestCommandTokenStoreTimeout(t *testing.T) {
	requireSh(t)
	ctx := context.Background()
	// A builtin loop keeps the test free of orphaned helper processes; the
	// process group kill also covers parents that spawn children.
	busy := []string{"sh", "-c", "while :; do :; done"}

	getStore, err := NewCommandTokenStore(busy, []string{"sh", "-c", `cat > /dev/null`}, nil, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}
	start := time.Now()
	if _, err := getStore.Get(ctx, "token-v2"); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Get() = %v, want timeout error", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("Get() took %s, want it to stop near the timeout", elapsed)
	}

	setStore, err := NewCommandTokenStore([]string{"sh", "-c", "exit 0"}, busy, nil, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}
	start = time.Now()
	if err := setStore.Set(ctx, "token-v2", &oauth2.Token{AccessToken: "secret"}); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Set() = %v, want timeout error", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("Set() took %s, want it to stop near the timeout", elapsed)
	}
}

func TestNewCommandTokenStoreRequiresBothCommands(t *testing.T) {
	if _, err := NewCommandTokenStore(nil, []string{"sh"}, nil, time.Second); err == nil {
		t.Fatal("NewCommandTokenStore(nil get) = nil error, want error")
	}
	if _, err := NewCommandTokenStore([]string{"sh"}, nil, nil, time.Second); err == nil {
		t.Fatal("NewCommandTokenStore(nil set) = nil error, want error")
	}
}

func TestNewCommandTokenStoreDefaultsTimeout(t *testing.T) {
	store, err := NewCommandTokenStore([]string{"sh"}, []string{"sh"}, nil, 0)
	if err != nil {
		t.Fatalf("NewCommandTokenStore() error = %v, want nil", err)
	}
	if store.timeout != defaultCommandTimeout {
		t.Fatalf("timeout = %s, want %s", store.timeout, defaultCommandTimeout)
	}
	if !store.anyExitIsNotFound {
		t.Fatal("anyExitIsNotFound = false, want true when no exit codes are configured")
	}
}

func TestCappedBufferLimitsAndKeepsUTF8(t *testing.T) {
	buffer := newCappedBuffer(5)
	if _, err := buffer.Write([]byte("abcdef")); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}
	if got := buffer.String(); got != "abcde…" {
		t.Fatalf("String() = %q, want %q", got, "abcde…")
	}

	// "a" + "é" (2 bytes) + a truncated "é" cannot be sliced by bytes.
	buffer = newCappedBuffer(4)
	if _, err := buffer.Write([]byte("aéé")); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}
	got := buffer.String()
	if !utf8.ValidString(got) {
		t.Fatalf("String() = %q, want valid UTF-8", got)
	}
	if got != "aé…" {
		t.Fatalf("String() = %q, want %q", got, "aé…")
	}
}

func TestTruncateKeepsValidUTF8(t *testing.T) {
	long := strings.Repeat("é", commandErrorLimit+10)
	got := truncate(long)
	if !utf8.ValidString(got) {
		t.Fatalf("truncate() = %q, want valid UTF-8", got)
	}
	if runes := utf8.RuneCountInString(got); runes != commandErrorLimit+1 {
		t.Fatalf("truncate() rune count = %d, want %d", runes, commandErrorLimit+1)
	}
}
