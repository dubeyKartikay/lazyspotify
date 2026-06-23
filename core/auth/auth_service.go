package auth

import (
	"context"
	"fmt"
	"net/http"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

type AuthService struct {
	sptAuth    *spotifyauth.Authenticator
	tknChannel chan *oauth2.Token
	authConfig *AuthConfig
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
		a.tknChannel <- tok
	}
	return callback, errCh
}

func (a *AuthService) GetSpotifyClient(tkn *oauth2.Token) *spotify.Client {
	httpClient := a.sptAuth.Client(context.Background(), tkn)
	return spotify.New(httpClient)
}

// OAuthHTTPClient returns an HTTP client that attaches the user's access token
// and refreshes it the same way as the official Spotify Web API client.
func (a *AuthService) OAuthHTTPClient(ctx context.Context, tkn *oauth2.Token) *http.Client {
	return a.sptAuth.Client(ctx, tkn)
}
