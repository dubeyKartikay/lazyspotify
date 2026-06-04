package auth

import (
	"errors"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestWrapKeyringErrorPreservesNilAndNotFound(t *testing.T) {
	if err := wrapKeyringError(nil); err != nil {
		t.Fatalf("wrapKeyringError(nil) = %v, want nil", err)
	}
	if err := wrapKeyringError(keyring.ErrNotFound); !errors.Is(err, keyring.ErrNotFound) {
		t.Fatalf("wrapKeyringError(ErrNotFound) = %v, want errors.Is ErrNotFound", err)
	}
}

func TestWrapKeyringErrorAddsActionableContext(t *testing.T) {
	root := errors.New("dbus unavailable")

	err := wrapKeyringError(root)
	if err == nil {
		t.Fatal("wrapKeyringError() = nil, want contextual error")
	}
	if !errors.Is(err, root) {
		t.Fatalf("wrapKeyringError() = %v, want to wrap root error", err)
	}
	if !strings.Contains(err.Error(), "system keyring unavailable") {
		t.Fatalf("wrapKeyringError() = %q, want system keyring context", err.Error())
	}
}
