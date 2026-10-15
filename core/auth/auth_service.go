package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

type AuthService struct {
	sptAuth        *spotifyauth.Authenticator
	tknChannel     chan *oauth2.Token
	authConfig     *AuthConfig
	tokenDelivered sync.Once
}

func NewAuthService(redirectURI string) *AuthService {
	authConfig := NewAuthConfig()
	sptAuth := spotifyauth.New(
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			"streaming",
			"app-remote-control",
			"playlist-read-private",
			"playlist-read-collaborative",
			"user-follow-read",
			"user-library-read",
			"user-read-private",
		),
		spotifyauth.WithClientID(authConfig.clientID),
	)
	return &AuthService{
		sptAuth:    sptAuth,
		tknChannel: make(chan *oauth2.Token, 1),
		authConfig: authConfig,
	}
}

func (a *AuthService) GetAuthURL() string {
	return a.sptAuth.AuthURL(a.authConfig.state,
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("code_challenge", a.authConfig.codeChallenge),
		oauth2.SetAuthURLParam("client_id", a.authConfig.clientID),
	)
}

func (a *AuthService) GetTokenChannel() chan *oauth2.Token {
	return a.tknChannel
}

func (a *AuthService) MakeOauthCallbackHandler() (func(w http.ResponseWriter, r *http.Request), chan error) {
	errCh := make(chan error, 1)
	callback := func(w http.ResponseWriter, r *http.Request) {
		tok, err := a.sptAuth.Token(r.Context(), a.authConfig.state, r, oauth2.SetAuthURLParam("code_verifier", a.authConfig.codeVerifier))
		if err != nil {
			http.Error(w, "Couldn't get token", http.StatusForbidden)
			errCh <- err
			return
		}
		if st := r.FormValue("state"); st != a.authConfig.state {
			http.NotFound(w, r)
			errCh <- fmt.Errorf("state mismatch: %s != %s", st, a.authConfig.state)
			return
		}
		_, _ = fmt.Fprintln(w, "Authentication successful. You can close this window.")
		a.sendToken(tok)
	}
	return callback, errCh
}

func (a *AuthService) GetSpotifyClient(tkn *oauth2.Token) *spotify.Client {
	httpClient := a.sptAuth.Client(context.Background(), tkn)
	return spotify.New(httpClient)
}

// ExchangePastedCallback completes the login from a callback value the user
// pasted back, for browsers that cannot reach the callback server.
func (a *AuthService) ExchangePastedCallback(ctx context.Context, pasted string) (*oauth2.Token, error) {
	code, err := a.authConfig.ParsePastedCallback(pasted)
	if err != nil {
		return nil, err
	}
	tok, err := a.sptAuth.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", a.authConfig.codeVerifier))
	if err != nil {
		return nil, fmt.Errorf("failed exchanging pasted code: %w", err)
	}
	a.sendToken(tok)
	return tok, nil
}

// sendToken lets either the browser callback or pasted callback complete this
// login once. A late result cannot leave another token queued after completion.
func (a *AuthService) sendToken(tok *oauth2.Token) {
	a.tokenDelivered.Do(func() { a.tknChannel <- tok })
}
