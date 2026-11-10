package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"golang.org/x/oauth2"
)

// TokenStore persists the Spotify OAuth token. Implementations are selected
// with auth.token-store.backend so that headless setups without a Secret
// Service keyring can still log in.
type TokenStore interface {
	// Get returns the stored token for key, or ErrTokenNotFound when the store
	// holds no usable token yet.
	Get(ctx context.Context, key string) (*oauth2.Token, error)
	// Set replaces the stored token for key.
	Set(ctx context.Context, key string, token *oauth2.Token) error
}

var (
	_ TokenStore = (*KeyringStore)(nil)
	_ TokenStore = (*FileTokenStore)(nil)
	_ TokenStore = (*CommandTokenStore)(nil)
)

// ErrTokenNotFound reports that the store holds no token for the requested
// key, which means the user still has to log in.
var ErrTokenNotFound = errors.New("no stored token found")

// errorTokenStore surfaces a token store construction error at first use.
type errorTokenStore struct{ err error }

func (s errorTokenStore) Get(context.Context, string) (*oauth2.Token, error) {
	return nil, s.err
}

func (s errorTokenStore) Set(context.Context, string, *oauth2.Token) error {
	return s.err
}

// NewTokenStore builds the token store described by the loaded app config.
func NewTokenStore() (TokenStore, error) {
	return NewTokenStoreFromConfig(utils.GetConfig())
}

// NewTokenStoreFromConfig builds the token store described by cfg.
func NewTokenStoreFromConfig(cfg utils.AppConfig) (TokenStore, error) {
	store := cfg.Auth.TokenStore
	switch utils.NormalizeTokenStoreBackend(store.Backend) {
	case utils.TokenStoreBackendKeyring:
		return NewKeyringStore(cfg.Auth.Keyring.Service), nil
	case utils.TokenStoreBackendFile:
		return NewFileTokenStore(store.File.Path)
	case utils.TokenStoreBackendCommand:
		return NewCommandTokenStore(
			store.Command.Get,
			store.Command.Set,
			store.Command.NotFoundExitCodes,
			time.Duration(store.Command.Timeout)*time.Second,
		)
	default:
		return nil, fmt.Errorf(
			"unsupported auth.token-store.backend %q; supported backends: %s",
			store.Backend,
			strings.Join(utils.SupportedTokenStoreBackends(), ", "),
		)
	}
}

// encodeToken serializes a token, refusing values that could never
// authenticate. decodeToken treats those same values as missing on read.
func encodeToken(token *oauth2.Token) (string, error) {
	if token == nil {
		return "", errors.New("refusing to store a nil token")
	}
	if token.AccessToken == "" && token.RefreshToken == "" {
		return "", errors.New("refusing to store a token without an access or refresh token")
	}
	serialized, err := json.Marshal(token)
	if err != nil {
		logger.Log.Error().Err(err).Msg("error marshaling token")
		return "", fmt.Errorf("encode token: %w", err)
	}
	return string(serialized), nil
}

// decodeToken parses a stored token. Anything that cannot authenticate — bad
// JSON, null, {} or a token with no access and no refresh token — reads as
// missing so the next login can replace it.
func decodeToken(key, serialized string) (*oauth2.Token, error) {
	token := oauth2.Token{}
	if err := json.Unmarshal([]byte(serialized), &token); err != nil {
		logger.Log.Warn().
			Err(err).
			Str("key", key).
			Msg("stored token is unreadable; treating it as missing so the next login can replace it")
		return nil, ErrTokenNotFound
	}
	if token.AccessToken == "" && token.RefreshToken == "" {
		logger.Log.Warn().
			Str("key", key).
			Msg("stored token is empty; treating it as missing so the next login can replace it")
		return nil, ErrTokenNotFound
	}
	return &token, nil
}
