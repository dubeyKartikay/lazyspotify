package auth

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/zalando/go-keyring"
)

// Keyring stores tokens in the system keyring through Secret Service on Linux
// and the platform equivalents elsewhere. It implements TokenStore.
type Keyring struct {
	service string
}

// NewKeyring builds a keyring store for the given keyring service name.
func NewKeyring(service string) *Keyring {
	return &Keyring{service: service}
}

func (k *Keyring) GetString(key string) (string, error) {
	value, err := keyring.Get(k.service, key)
	if err != nil {
		return "", wrapKeyringError(err)
	}
	return value, nil
}

func (k *Keyring) SetString(key, value string) error {
	return wrapKeyringError(keyring.Set(k.service, key, value))
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
