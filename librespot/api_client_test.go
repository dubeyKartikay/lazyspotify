package librespot

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

func newTestLibrespotClient(t *testing.T, handler http.Handler) (*LibrespotApiClient, func()) {
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

	return &LibrespotApiClient{
		server: NewLibrespotApiServer(host, port),
		client: server.Client(),
	}, server.Close
}

func TestPlayPostsExpectedJSONBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != playPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, playPath)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}

		var request models.PlayRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("Decode PlayRequest error = %v, want nil", err)
		}
		if request.Uri != "spotify:album:album-id" {
			t.Fatalf("request.Uri = %q, want spotify:album:album-id", request.Uri)
		}
		if request.SkipToUri != "spotify:track:track-id" {
			t.Fatalf("request.SkipToUri = %q, want spotify:track:track-id", request.SkipToUri)
		}
		if request.Paused {
			t.Fatal("request.Paused = true, want false")
		}

		w.WriteHeader(http.StatusAccepted)
	})

	client, cleanup := newTestLibrespotClient(t, handler)
	defer cleanup()

	status := client.Play(context.Background(), "spotify:album:album-id", "spotify:track:track-id", false)
	if status != http.StatusAccepted {
		t.Fatalf("Play() = %d, want %d", status, http.StatusAccepted)
	}
}

func TestResolvePlaylistTracksNormalizesPaginationAndDecodesResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != resolveTracksPath {
			t.Fatalf("path = %q, want %q", r.URL.Path, resolveTracksPath)
		}

		query := r.URL.Query()
		assertQuery(t, query.Get("uri"), "spotify:playlist:playlist-id", "uri")
		assertQuery(t, query.Get("offset"), "0", "offset")
		assertQuery(t, query.Get("limit"), "10", "limit")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"uri": "spotify:playlist:playlist-id",
			"offset": 0,
			"limit": 10,
			"total": 11,
			"has_next": true,
			"tracks": [{
				"uri": "spotify:track:track-id",
				"uid": "track-id",
				"name": "Track",
				"artists": ["Artist"],
				"album_name": "Album",
				"duration_ms": 180000,
				"album_uri": "spotify:album:album-id",
				"artist_uri": "spotify:artist:artist-id",
				"metadata": {"disc_number": "1"}
			}]
		}`))
	})

	client, cleanup := newTestLibrespotClient(t, handler)
	defer cleanup()

	got, err := client.ResolvePlaylistTracks(context.Background(), "spotify:playlist:playlist-id", -20, 0)
	if err != nil {
		t.Fatalf("ResolvePlaylistTracks() error = %v, want nil", err)
	}
	if got.URI != "spotify:playlist:playlist-id" {
		t.Fatalf("response.URI = %q, want spotify:playlist:playlist-id", got.URI)
	}
	if got.Offset != 0 || got.Limit != 10 || got.Total != 11 || !got.HasNext {
		t.Fatalf("pagination = offset %d limit %d total %d hasNext %t, want 0/10/11/true", got.Offset, got.Limit, got.Total, got.HasNext)
	}
	if len(got.Tracks) != 1 {
		t.Fatalf("len(response.Tracks) = %d, want 1", len(got.Tracks))
	}
	track := got.Tracks[0]
	if track.URI != "spotify:track:track-id" || track.Name != "Track" || track.Metadata["disc_number"] != "1" {
		t.Fatalf("decoded track = %#v, want track-id/Track metadata", track)
	}
}

func TestResolvePlaylistTracksReportsUnavailableResolverEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tracks": []}`))
	})

	client, cleanup := newTestLibrespotClient(t, handler)
	defer cleanup()

	got, err := client.ResolvePlaylistTracks(context.Background(), "spotify:playlist:playlist-id", 0, 10)
	if err == nil {
		t.Fatal("ResolvePlaylistTracks() error = nil, want unavailable endpoint error")
	}
	if got != nil {
		t.Fatalf("ResolvePlaylistTracks() response = %#v, want nil", got)
	}
	if !strings.Contains(err.Error(), "not available") {
		t.Fatalf("ResolvePlaylistTracks() error = %q, want unavailable endpoint error", err.Error())
	}
}

func TestGetVolumeRejectsErrorStatusWithoutDecodingBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != volumePath {
			t.Fatalf("path = %q, want %q", r.URL.Path, volumePath)
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`not-json`))
	})

	client, cleanup := newTestLibrespotClient(t, handler)
	defer cleanup()

	got, err := client.GetVolume(context.Background())
	if err == nil {
		t.Fatal("GetVolume() error = nil, want status error")
	}
	if got != nil {
		t.Fatalf("GetVolume() = %#v, want nil", got)
	}
	if !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("GetVolume() error = %q, want status 400", err.Error())
	}
}

func TestDoWithRetryRetriesServerErrorsAndReplaysRequestBody(t *testing.T) {
	const requestBody = `{"volume":123,"relative":false}`

	var attempts int
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("io.ReadAll(body) error = %v, want nil", err)
		}
		bodies = append(bodies, string(body))

		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v, want nil", err)
	}

	resp, err := DoWithRetry(server.Client(), req, 1, time.Nanosecond)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v, want nil", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("response status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if len(bodies) != 2 || bodies[0] != requestBody || bodies[1] != requestBody {
		t.Fatalf("request bodies = %#v, want two copies of %q", bodies, requestBody)
	}
}

func TestDoWithRetryDoesNotRetryClientErrors(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v, want nil", err)
	}

	resp, err := DoWithRetry(server.Client(), req, 3, time.Nanosecond)
	if err != nil {
		t.Fatalf("DoWithRetry() error = %v, want nil for non-retriable client status", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("response status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func assertQuery(t *testing.T, got string, want string, name string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s query = %q, want %q", name, got, want)
	}
}
