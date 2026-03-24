package spotify

import (
	"context"
	"errors"
	"net/http"

	"github.com/dubeyKartikay/lazyspotify/core/auth"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/zalando/go-keyring"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

type SpotifyClient struct {
  client *spotify.Client
}

func NewSpotifyClient(ctx context.Context,auth *auth.Authenticator) (*SpotifyClient,error) {
	client,err:= auth.GetClient(ctx)
	if(err != nil) {
		logger.Log.Error().Err(err).Msg("error getting spotify client")
		return nil, err
  }
  return &SpotifyClient{
    client: client,
  },nil
}

func (s *SpotifyClient) GetUserID(ctx context.Context) (string, error) {
  user, err := s.client.CurrentUser(ctx)
	if(err != nil) {
    logger.Log.Error().Err(err).Msg("error getting user id")
    return "", err
	}
  return user.ID, nil
}

func IsAuthError(err error) bool {
	var spotifyErr spotify.Error
	if(errors.Is(err,keyring.ErrNotFound)) {
  	return true
	}
	if errors.As(err, &spotifyErr) && spotifyErr.Status == http.StatusUnauthorized {
			return true
	}
	var retrieveErr *oauth2.RetrieveError
	return errors.As(err, &retrieveErr) && retrieveErr.ErrorCode == "invalid_grant"
}

