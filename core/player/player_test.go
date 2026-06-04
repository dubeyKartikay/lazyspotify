package player

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/dubeyKartikay/lazyspotify/librespot"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

func newTestPlayer(t *testing.T, handler http.Handler) (*Player, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v, want nil", server.URL, err)
	}
	host, rawPort, err := net.SplitHostPort(parsed.Host)
	if err != nil {
		t.Fatalf("net.SplitHostPort(%q) error = %v, want nil", parsed.Host, err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatalf("strconv.Atoi(%q) error = %v, want nil", rawPort, err)
	}

	apiServer := librespot.NewLibrespotApiServer(host, port)
	return &Player{
		librespot: &librespot.Librespot{
			Client: librespot.NewLibrespotApiClient(apiServer),
		},
	}, server.Close
}

func TestPlayTrackUsesContextURIAndSkipToURIForPlaylistPlayback(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/player/play" {
			t.Fatalf("path = %q, want /player/play", r.URL.Path)
		}

		var request models.PlayRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("Decode PlayRequest error = %v, want nil", err)
		}
		if request.Uri != "spotify:playlist:playlist-id" {
			t.Fatalf("request.Uri = %q, want spotify:playlist:playlist-id", request.Uri)
		}
		if request.SkipToUri != "spotify:track:track-id" {
			t.Fatalf("request.SkipToUri = %q, want spotify:track:track-id", request.SkipToUri)
		}
		if request.Paused {
			t.Fatal("request.Paused = true, want false")
		}

		w.WriteHeader(http.StatusNoContent)
	})

	player, cleanup := newTestPlayer(t, handler)
	defer cleanup()

	if err := player.PlayTrack(context.Background(), "spotify:track:track-id", "spotify:playlist:playlist-id"); err != nil {
		t.Fatalf("PlayTrack() error = %v, want nil", err)
	}
}

func TestPlayTrackUsesTrackURIWhenContextIsNotAPlayableCollection(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request models.PlayRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("Decode PlayRequest error = %v, want nil", err)
		}
		if request.Uri != "spotify:track:track-id" {
			t.Fatalf("request.Uri = %q, want spotify:track:track-id", request.Uri)
		}
		if request.SkipToUri != "" {
			t.Fatalf("request.SkipToUri = %q, want empty string", request.SkipToUri)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	player, cleanup := newTestPlayer(t, handler)
	defer cleanup()

	if err := player.PlayTrack(context.Background(), "spotify:track:track-id", "spotify:artist:artist-id"); err != nil {
		t.Fatalf("PlayTrack() error = %v, want nil", err)
	}
}

func TestPlayTrackReturnsDaemonStatusErrors(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	})

	player, cleanup := newTestPlayer(t, handler)
	defer cleanup()

	if err := player.PlayTrack(context.Background(), "spotify:track:track-id", ""); err == nil {
		t.Fatal("PlayTrack() error = nil, want daemon status error")
	}
}

func TestShuffleUpdatesCachedStateOnlyAfterSuccessfulDaemonResponse(t *testing.T) {
	var status = http.StatusNoContent
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/player/shuffle_context" {
			t.Fatalf("path = %q, want /player/shuffle_context", r.URL.Path)
		}

		var request models.ShuffleRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("Decode ShuffleRequest error = %v, want nil", err)
		}
		if !request.Shuffle {
			t.Fatal("request.Shuffle = false, want true")
		}

		w.WriteHeader(status)
	})

	player, cleanup := newTestPlayer(t, handler)
	defer cleanup()

	if err := player.Shuffle(context.Background(), true); err != nil {
		t.Fatalf("Shuffle(true) error = %v, want nil", err)
	}
	if !player.Shuffled() {
		t.Fatal("Shuffled() = false, want true after successful shuffle")
	}

	status = http.StatusInternalServerError
	if err := player.Shuffle(context.Background(), true); err == nil {
		t.Fatal("Shuffle(true) error = nil, want daemon status error")
	}
	if !player.Shuffled() {
		t.Fatal("Shuffled() = false, want previous successful state preserved")
	}
}
