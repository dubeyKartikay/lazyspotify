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
			"app-remote-control",
			spotifyauth.ScopeUserReadPrivate,
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
			spotifyauth.ScopePlaylistReadCollaborative,
			spotifyauth.ScopeUserFollowModify,
			spotifyauth.ScopeUserFollowRead,
			spotifyauth.ScopeUserLibraryModify,
			spotifyauth.ScopeUserLibraryRead,
			spotifyauth.ScopeUserReadCurrentlyPlaying,
			spotifyauth.ScopeUserReadPlaybackState,
			spotifyauth.ScopeUserModifyPlaybackState,
			spotifyauth.ScopeUserReadRecentlyPlayed,
			spotifyauth.ScopeUserTopRead,
			spotifyauth.ScopeUserReadEmail,
			spotifyauth.ScopeStreaming,
		),
		spotifyauth.WithClientID("565c1a413de9452da373f1ed3aa6afbe"),
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
		oauth2.SetAuthURLParam("client_id", "565c1a413de9452da373f1ed3aa6afbe"),
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
		}
		if st := r.FormValue("state"); st != a.authConfig.state {
			http.NotFound(w, r)
			errCh <- fmt.Errorf("state mismatch: %s != %s", st, a.authConfig.state)
		}
		a.tknChannel <- tok
	}
	return callback, errCh
}

func (a *AuthService) GetSpotifyClient(tkn *oauth2.Token) *spotify.Client {
	httpClient := a.sptAuth.Client(context.Background(), tkn)
	return spotify.New(httpClient)
}
