package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

type Authenticator struct {
	AuthServer  *AuthServer
	authService *AuthService
	tokenStore  TokenStore
	tokenKey    string
	oauthErrCh  chan error
}

func New() *Authenticator {
	authServer := NewAuthServer()
	authService := NewAuthService(authServer.GetOauthRedirectURI())
	oauthCallbackFunc, oauthErr := authService.MakeOauthCallbackHandler()
	authServer.InitAuthServer(oauthCallbackFunc)
	tokenStore, storeErr := NewTokenStore()
	if storeErr != nil {
		// The UI builds the authenticator before it can report an error, so
		// keep the failure in the store: the first token read at startup
		// surfaces it before the browser login starts.
		logger.Log.Error().Err(storeErr).Msg("failed to configure token store")
		tokenStore = errorTokenStore{err: storeErr}
	}
	return &Authenticator{
		AuthServer:  authServer,
		tokenStore:  tokenStore,
		authService: authService,
		tokenKey:    utils.GetConfig().AuthTokenKey(),
		oauthErrCh:  oauthErr,
	}
}

func (a *Authenticator) GetAuthToken(ctx context.Context) (*oauth2.Token, error) {
	tkn, err := a.tokenStore.Get(ctx, a.tokenKey)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			logger.Log.Debug().Msg("no stored token; login required")
			return nil, err
		}
		logger.Log.Error().Err(err).Msg("error getting token")
		return nil, err
	}
	return tkn, nil
}

func (a *Authenticator) GetClient(ctx context.Context) (*spotify.Client, error) {
	tkn, err := a.GetAuthToken(ctx)
	if err != nil {
		return nil, err
	}
	return a.authService.GetSpotifyClient(tkn), nil
}

func (a *Authenticator) ReAuthenticate(ctx context.Context, updates chan<- string) (*oauth2.Token, error) {
	logger.Log.Info().Msg("authenticating with spotify")
	severErrCh := a.AuthServer.Start()
	defer a.AuthServer.Shutdown()
	var tkn *oauth2.Token
	updates <- "awaiting authentication"
	select {
	case err := <-a.oauthErrCh:
		return nil, err
	case err := <-severErrCh:
		return nil, err
	case tkn = <-a.authService.GetTokenChannel():
	}
	if tkn == nil {
		return nil, fmt.Errorf("authentication failed: received empty token")
	}
	if err := a.saveToken(ctx, tkn); err != nil {
		return nil, fmt.Errorf("failed to save authentication token: %w", err)
	}
	updates <- "success"
	return tkn, nil
}

func (a *Authenticator) GetAuthURL() string {
	return a.authService.GetAuthURL()
}

// SetPostAuthRedirect registers a provider for a follow-up URL the browser
// should be sent to after a successful login; see AuthService.SetPostAuthRedirect.
func (a *Authenticator) SetPostAuthRedirect(provider func() (url string, code string, ok bool)) {
	a.authService.SetPostAuthRedirect(provider)
}

// CompleteLoginWithPastedCallback finishes the login from a callback value the
// user pasted back, for browsers that cannot reach the callback server.
func (a *Authenticator) CompleteLoginWithPastedCallback(ctx context.Context, pasted string) error {
	_, err := a.authService.ExchangePastedCallback(ctx, pasted)
	return err
}

func (a *Authenticator) saveToken(ctx context.Context, token *oauth2.Token) error {
	return a.tokenStore.Set(ctx, a.tokenKey, token)
}
