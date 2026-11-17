package auth

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// KeyringStore stores tokens in the system keyring through Secret Service on
// Linux and the platform equivalents elsewhere.
type KeyringStore struct {
	service string
}

// NewKeyringStore builds a keyring store for the given keyring service name.
func NewKeyringStore(service string) *KeyringStore {
	return &KeyringStore{service: service}
}

func (k *KeyringStore) Get(ctx context.Context, key string) (*oauth2.Token, error) {
	_ = ctx
	value, err := keyring.Get(k.service, key)
	if err != nil {
		return nil, wrapKeyringError(err)
	}
	return decodeToken(key, value)
}

func (k *KeyringStore) Set(ctx context.Context, key string, token *oauth2.Token) error {
	_ = ctx
	encoded, err := encodeToken(token)
	if err != nil {
		return err
	}
	return wrapKeyringError(keyring.Set(k.service, key, encoded))
}

func wrapKeyringError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrTokenNotFound
	}

	if runtime.GOOS == "linux" {
		return fmt.Errorf(
			"system keyring unavailable: %w; start a Secret Service provider (for example gnome-keyring or KWallet) or set auth.token-store.backend to \"file\" or \"command\"",
			err,
		)
	}

	return fmt.Errorf(
		"system keyring unavailable: %w; set auth.token-store.backend to \"file\" or \"command\" to store tokens another way",
		err,
	)
}
