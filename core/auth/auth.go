package auth

import (
	"context"
	"fmt"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

type Authenticator struct {
	AuthServer  *AuthServer
	authService *AuthService
	keyring     *Keyring
	tokenKey    string
	oauthErrCh  chan error
}

func New() *Authenticator {
	authServer := NewAuthServer()
	authService := NewAuthService(authServer.GetOauthRedirectURI())
	oauthCallbackFunc, err := authService.MakeOauthCallbackHandler()
	authServer.InitAuthServer(oauthCallbackFunc)
	return &Authenticator{
		AuthServer:  authServer,
		keyring:     NewSpotifyKeyring(),
		authService: authService,
		tokenKey:    utils.GetConfig().Auth.Keyring.Key,
		oauthErrCh:  err,
	}
}

func (a *Authenticator) GetAuthToken(ctx context.Context) (*oauth2.Token, error) {
	tkn, err := a.keyring.GetToken(a.tokenKey)
	if err != nil {
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
	if err := a.saveToken(tkn); err != nil {
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

func (a *Authenticator) saveToken(token *oauth2.Token) error {
	return a.keyring.SetToken(a.tokenKey, token)
}

// CompleteLoginWithPastedCallback finishes the login from a callback value the
// user pasted back, for browsers that cannot reach the callback server.
func (a *Authenticator) CompleteLoginWithPastedCallback(ctx context.Context, pasted string) error {
	_, err := a.authService.ExchangePastedCallback(ctx, pasted)
	return err
}
