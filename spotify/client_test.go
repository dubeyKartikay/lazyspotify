package spotify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
	spotifyapi "github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2"
)

func newTestClient(t *testing.T, handler http.Handler) (*SpotifyClient, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	client := spotifyapi.New(server.Client(), spotifyapi.WithBaseURL(server.URL+"/"))

	return &SpotifyClient{client: client}, server.Close
}

func TestGetPlaylistTracksRequestsParsedPlaylistIDAndFiltersUnavailableItems(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/playlists/playlist-id/tracks" {
			t.Fatalf("path = %q, want /playlists/playlist-id/tracks", r.URL.Path)
		}

		query := r.URL.Query()
		assertQueryValue(t, query.Get("offset"), "20", "offset")
		assertQueryValue(t, query.Get("limit"), "10", "limit")
		assertQueryValue(t, query.Get("additional_types"), "episode,track", "additional_types")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"href": "https://api.spotify.test/playlists/playlist-id/tracks",
			"limit": 10,
			"offset": 20,
			"total": 3,
			"items": [
				{"track": {"type": "track", "id": "track-1", "name": "First", "uri": "spotify:track:track-1"}},
				{"track": null},
				{"track": {"type": "track", "id": "track-2", "name": "Second", "uri": "spotify:track:track-2"}}
			]
		}`))
	})

	client, cleanup := newTestClient(t, handler)
	defer cleanup()

	tracks, err := client.GetPlaylistTracks(context.Background(), "spotify:playlist:playlist-id", 20)
	if err != nil {
		t.Fatalf("GetPlaylistTracks() error = %v, want nil", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("len(tracks) = %d, want 2", len(tracks))
	}
	if tracks[0].URI != "spotify:track:track-1" || tracks[1].URI != "spotify:track:track-2" {
		t.Fatalf("track URIs = [%q, %q], want [spotify:track:track-1, spotify:track:track-2]", tracks[0].URI, tracks[1].URI)
	}
}

func TestGetPlaylistTracksRejectsInvalidURIWithoutCallingAPI(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Fatalf("unexpected API call to %s", r.URL.String())
	})

	client, cleanup := newTestClient(t, handler)
	defer cleanup()

	tracks, err := client.GetPlaylistTracks(context.Background(), "playlist-id", 0)
	if err == nil {
		t.Fatal("GetPlaylistTracks() error = nil, want invalid URI error")
	}
	if tracks != nil {
		t.Fatalf("tracks = %#v, want nil", tracks)
	}
	if called {
		t.Fatal("API was called for an invalid URI")
	}
}

func TestGetFirstSavedTrackReturnsFirstSavedTrackURI(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/tracks" {
			t.Fatalf("path = %q, want /me/tracks", r.URL.Path)
		}
		query := r.URL.Query()
		assertQueryValue(t, query.Get("offset"), "0", "offset")
		assertQueryValue(t, query.Get("limit"), "10", "limit")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"href": "https://api.spotify.test/me/tracks",
			"limit": 10,
			"offset": 0,
			"total": 2,
			"items": [
				{"added_at": "2026-01-01T00:00:00Z", "track": {"type": "track", "id": "first", "name": "First", "uri": "spotify:track:first"}},
				{"added_at": "2026-01-02T00:00:00Z", "track": {"type": "track", "id": "second", "name": "Second", "uri": "spotify:track:second"}}
			]
		}`))
	})

	client, cleanup := newTestClient(t, handler)
	defer cleanup()

	uri, err := client.GetFirstSavedTrack(context.Background())
	if err != nil {
		t.Fatalf("GetFirstSavedTrack() error = %v, want nil", err)
	}
	if uri != "spotify:track:first" {
		t.Fatalf("GetFirstSavedTrack() = %q, want spotify:track:first", uri)
	}
}

func TestGetFirstSavedTrackErrorsWhenLibraryIsEmpty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"limit": 10, "offset": 0, "total": 0, "items": []}`))
	})

	client, cleanup := newTestClient(t, handler)
	defer cleanup()

	uri, err := client.GetFirstSavedTrack(context.Background())
	if err == nil {
		t.Fatal("GetFirstSavedTrack() error = nil, want empty library error")
	}
	if uri != "" {
		t.Fatalf("GetFirstSavedTrack() = %q, want empty URI", uri)
	}
	if !strings.Contains(err.Error(), "no saved tracks found") {
		t.Fatalf("GetFirstSavedTrack() error = %q, want no saved tracks error", err.Error())
	}
}

func TestSearchTracksReturnsEmptyPageWhenSpotifyOmitsTracksField(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Fatalf("path = %q, want /search", r.URL.Path)
		}
		query := r.URL.Query()
		assertQueryValue(t, query.Get("q"), "blue train", "q")
		assertQueryValue(t, query.Get("type"), "track", "type")
		assertQueryValue(t, query.Get("offset"), "30", "offset")
		assertQueryValue(t, query.Get("limit"), "15", "limit")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	client, cleanup := newTestClient(t, handler)
	defer cleanup()

	page, err := client.SearchTracks(context.Background(), "blue train", 30, 15)
	if err != nil {
		t.Fatalf("SearchTracks() error = %v, want nil", err)
	}
	if page == nil {
		t.Fatal("SearchTracks() = nil, want empty page")
	}
	if len(page.Tracks) != 0 {
		t.Fatalf("len(page.Tracks) = %d, want 0", len(page.Tracks))
	}
}

func TestIDFromURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    string
		wantErr bool
	}{
		{name: "track", uri: "spotify:track:abc123", want: "abc123"},
		{name: "playlist", uri: "spotify:user:user-id:playlist:playlist-id", want: "playlist-id"},
		{name: "missing prefix parts", uri: "abc123", wantErr: true},
		{name: "empty id", uri: "spotify:track:", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := idFromURI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("idFromURI(%q) error = nil, want error", tt.uri)
				}
				return
			}
			if err != nil {
				t.Fatalf("idFromURI(%q) error = %v, want nil", tt.uri, err)
			}
			if got != tt.want {
				t.Fatalf("idFromURI(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "missing keyring token", err: keyring.ErrNotFound, want: true},
		{name: "spotify unauthorized", err: spotifyapi.Error{Status: http.StatusUnauthorized, Message: "unauthorized"}, want: true},
		{name: "spotify forbidden", err: spotifyapi.Error{Status: http.StatusForbidden, Message: "forbidden"}, want: false},
		{name: "oauth invalid grant", err: &oauth2.RetrieveError{ErrorCode: "invalid_grant"}, want: true},
		{name: "oauth transient error", err: &oauth2.RetrieveError{ErrorCode: "temporarily_unavailable"}, want: false},
		{name: "generic error", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthError(tt.err); got != tt.want {
				t.Fatalf("IsAuthError(%T) = %t, want %t", tt.err, got, tt.want)
			}
		})
	}
}

func assertQueryValue(t *testing.T, got string, want string, name string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s query = %q, want %q", name, got, want)
	}
}
