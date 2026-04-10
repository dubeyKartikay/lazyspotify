package spotify

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dubeyKartikay/lazyspotify/core/auth"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/zalando/go-keyring"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

type SpotifyClient struct {
	client *spotify.Client
}

func NewSpotifyClient(ctx context.Context, auth *auth.Authenticator) (*SpotifyClient, error) {
	client, err := auth.GetClient(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Msg("error getting spotify client")
		return nil, err
	}
	return &SpotifyClient{client: client}, nil
}

func (s *SpotifyClient) GetUserID(ctx context.Context) (string, error) {
	user, err := s.client.CurrentUser(ctx)
	if err != nil {
		logger.Log.Error().Err(err).Msg("error getting user id")
		return "", err
	}
	return user.ID, nil
}

func (s *SpotifyClient) GetFirstSavedTrack(ctx context.Context) (string, error) {
	tracks, err := s.GetSavedTracks(ctx, 0)
	if err != nil || tracks == nil || len(tracks.Tracks) == 0 {
		logger.Log.Error().Stack().Err(err).Msg("error getting daily mix")
		if err == nil {
			err = fmt.Errorf("no saved tracks found")
		}
		return "", err
	}
	logger.Log.Info().Any("playlists", tracks)
	return string(tracks.Tracks[0].URI), nil
}

func (s *SpotifyClient) GetUserPlaylists(ctx context.Context, offset int) (*spotify.SimplePlaylistPage, error) {
	list, err := s.client.CurrentUsersPlaylists(ctx, spotify.Offset(offset), spotify.Limit(10))
	if err != nil {
		logger.Log.Error().Err(err).Msg("error getting user playlists")
		return nil, err
	}
	return list, nil
}

func (s *SpotifyClient) GetSavedTracks(ctx context.Context, offset int) (*spotify.SavedTrackPage, error) {
	tracks, err := s.client.CurrentUsersTracks(ctx, spotify.Offset(offset), spotify.Limit(10))
	if err != nil {
		logger.Log.Error().Err(err).Msg("error getting saved tracks")
		return nil, err
	}
	return tracks, nil
}

func (s *SpotifyClient) GetSavedAlbums(ctx context.Context, offset int) (*spotify.SavedAlbumPage, error) {
	albums, err := s.client.CurrentUsersAlbums(ctx, spotify.Offset(offset), spotify.Limit(10))
	if err != nil {
		logger.Log.Error().Err(err).Msg("error getting saved albums")
		return nil, err
	}
	return albums, nil
}

func (s *SpotifyClient) GetFollowedArtists(ctx context.Context, after string) (*spotify.FullArtistCursorPage, error) {
	opts := []spotify.RequestOption{spotify.Limit(10)}
	if after != "" {
		opts = append(opts, spotify.After(after))
	}
	artists, err := s.client.CurrentUsersFollowedArtists(ctx, opts...)
	if err != nil {
		logger.Log.Error().Err(err).Msg("error getting followed artists")
		return nil, err
	}
	return artists, nil
}

func (s *SpotifyClient) GetPlaylistTracks(ctx context.Context, uri string, offset int) ([]spotify.FullTrack, error) {
	id, err := idFromURI(uri)
	if err != nil {
		return nil, err
	}
	page, err := s.client.GetPlaylistItems(ctx, spotify.ID(id), spotify.Offset(offset), spotify.Limit(10))
	if err != nil {
		logger.Log.Error().Err(err).Str("uri", uri).Int("offset", offset).Msg("error getting playlist tracks")
		return nil, err
	}
	tracks := make([]spotify.FullTrack, 0, len(page.Items))
	for _, item := range page.Items {
		if item.Track.Track == nil {
			continue
		}
		tracks = append(tracks, *item.Track.Track)
	}
	return tracks, nil
}

func (s *SpotifyClient) GetArtistAlbums(ctx context.Context, uri string, offset int) (*spotify.SimpleAlbumPage, error) {
	id, err := idFromURI(uri)
	if err != nil {
		return nil, err
	}
	page, err := s.client.GetArtistAlbums(ctx, spotify.ID(id), nil, spotify.Offset(offset), spotify.Limit(10))
	if err != nil {
		logger.Log.Error().Err(err).Str("uri", uri).Int("offset", offset).Msg("error getting artist albums")
		return nil, err
	}
	return page, nil
}

func (s *SpotifyClient) GetAlbumTracks(ctx context.Context, uri string, offset int) (*spotify.SimpleTrackPage, error) {
	id, err := idFromURI(uri)
	if err != nil {
		return nil, err
	}
	page, err := s.client.GetAlbumTracks(ctx, spotify.ID(id), spotify.Offset(offset), spotify.Limit(50))
	if err != nil {
		logger.Log.Error().Err(err).Str("uri", uri).Int("offset", offset).Msg("error getting album tracks")
		return nil, err
	}
	return page, nil
}

func (s *SpotifyClient) SearchPlaylists(ctx context.Context, query string, offset int, limit int) (*spotify.SimplePlaylistPage, error) {
	result, err := s.client.Search(ctx, query, spotify.SearchTypePlaylist, spotify.Offset(offset), spotify.Limit(limit))
	if err != nil {
		logger.Log.Error().Err(err).Str("query", query).Int("offset", offset).Msg("error searching playlists")
		return nil, err
	}
	if result.Playlists == nil {
		return &spotify.SimplePlaylistPage{}, nil
	}
	return result.Playlists, nil
}

func (s *SpotifyClient) SearchTracks(ctx context.Context, query string, offset int, limit int) (*spotify.FullTrackPage, error) {
	result, err := s.client.Search(ctx, query, spotify.SearchTypeTrack, spotify.Offset(offset), spotify.Limit(limit))
	if err != nil {
		logger.Log.Error().Err(err).Str("query", query).Int("offset", offset).Msg("error searching tracks")
		return nil, err
	}
	if result.Tracks == nil {
		return &spotify.FullTrackPage{}, nil
	}
	return result.Tracks, nil
}

func (s *SpotifyClient) SearchAlbums(ctx context.Context, query string, offset int, limit int) (*spotify.SimpleAlbumPage, error) {
	result, err := s.client.Search(ctx, query, spotify.SearchTypeAlbum, spotify.Offset(offset), spotify.Limit(limit))
	if err != nil {
		logger.Log.Error().Err(err).Str("query", query).Int("offset", offset).Msg("error searching albums")
		return nil, err
	}
	if result.Albums == nil {
		return &spotify.SimpleAlbumPage{}, nil
	}
	return result.Albums, nil
}

func (s *SpotifyClient) SearchArtists(ctx context.Context, query string, offset int, limit int) (*spotify.FullArtistPage, error) {
	result, err := s.client.Search(ctx, query, spotify.SearchTypeArtist, spotify.Offset(offset), spotify.Limit(limit))
	if err != nil {
		logger.Log.Error().Err(err).Str("query", query).Int("offset", offset).Msg("error searching artists")
		return nil, err
	}
	if result.Artists == nil {
		return &spotify.FullArtistPage{}, nil
	}
	return result.Artists, nil
}

func idFromURI(uri string) (string, error) {
	parts := strings.Split(uri, ":")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid spotify uri: %s", uri)
	}
	id := parts[len(parts)-1]
	if id == "" {
		return "", fmt.Errorf("invalid spotify uri: %s", uri)
	}
	return id, nil
}

func IsAuthError(err error) bool {
	var spotifyErr spotify.Error
	if errors.Is(err, keyring.ErrNotFound) {
		return true
	}
	if errors.As(err, &spotifyErr) && spotifyErr.Status == http.StatusUnauthorized {
		return true
	}
	var retrieveErr *oauth2.RetrieveError
	return errors.As(err, &retrieveErr) && retrieveErr.ErrorCode == "invalid_grant"
}
