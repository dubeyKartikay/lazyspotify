package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"golang.org/x/oauth2"
)

// TokenStore persists the serialized Spotify OAuth token. Implementations are
// selected with auth.token-store.backend so that headless setups without a
// Secret Service keyring can still log in.
type TokenStore interface {
	GetString(key string) (string, error)
	SetString(key, value string) error
}

// ErrTokenNotFound reports that the store holds no token for the requested
// key, which means the user still has to log in.
var ErrTokenNotFound = errors.New("no stored token found")

// NewTokenStore builds the token store described by the loaded app config.
func NewTokenStore() (TokenStore, error) {
	return NewTokenStoreFromConfig(utils.GetConfig())
}

// NewTokenStoreFromConfig builds the token store described by cfg.
func NewTokenStoreFromConfig(cfg utils.AppConfig) (TokenStore, error) {
	store := cfg.Auth.TokenStore
	switch utils.NormalizeTokenStoreBackend(store.Backend) {
	case utils.TokenStoreBackendKeyring:
		return NewKeyring(cfg.Auth.Keyring.Service), nil
	case utils.TokenStoreBackendFile:
		return NewFileTokenStore(store.File.Path)
	case utils.TokenStoreBackendCommand:
		return NewCommandTokenStore(
			store.Command.Get,
			store.Command.Set,
			time.Duration(store.Command.Timeout)*time.Second,
		)
	default:
		return nil, fmt.Errorf(
			"unsupported auth.token-store.backend %q; supported backends: keyring, file, command",
			store.Backend,
		)
	}
}

// GetOAuthToken reads and decodes the token stored under key.
func GetOAuthToken(store TokenStore, key string) (*oauth2.Token, error) {
	serialized, err := store.GetString(key)
	if err != nil {
		return nil, err
	}
	token := oauth2.Token{}
	if err := json.Unmarshal([]byte(serialized), &token); err != nil {
		return nil, fmt.Errorf("decode stored token: %w", err)
	}
	return &token, nil
}

// SetOAuthToken encodes and stores token under key.
func SetOAuthToken(store TokenStore, key string, token *oauth2.Token) error {
	serialized, err := json.Marshal(token)
	if err != nil {
		logger.Log.Error().Err(err).Msg("error marshaling token")
		return err
	}
	return store.SetString(key, string(serialized))
}
