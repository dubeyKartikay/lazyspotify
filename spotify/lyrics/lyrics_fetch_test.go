package lyrics

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchForbiddenSentinel(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	prev := colorLyricsBase
	colorLyricsBase = srv.URL + "/color-lyrics/v2/track/"
	t.Cleanup(func() { colorLyricsBase = prev })

	tr, err := Fetch(context.Background(), srv.Client(), "spotify:track:abcXYZ")
	if tr != nil {
		t.Fatalf("expected nil track, got %+v", tr)
	}
	if !errors.Is(err, ErrSpotifyForbidden) {
		t.Fatalf("errors.Is(ErrSpotifyForbidden) = false, err=%v", err)
	}
}

func TestFetchWithLRCLIBFallbackUsesLRCLIBOn403(t *testing.T) {
	t.Parallel()
	spotifySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	t.Cleanup(spotifySrv.Close)

	lrclibSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"syncedLyrics":"[00:01.00]hello\n[00:03.50]world","instrumental":false}`))
	}))
	t.Cleanup(lrclibSrv.Close)

	prevS := colorLyricsBase
	prevG := lrclibGetURL
	prevSearch := lrclibSearchURL
	colorLyricsBase = spotifySrv.URL + "/color-lyrics/v2/track/"
	lrclibGetURL = lrclibSrv.URL + "/api/get"
	lrclibSearchURL = lrclibSrv.URL + "/api/search"
	t.Cleanup(func() {
		colorLyricsBase = prevS
		lrclibGetURL = prevG
		lrclibSearchURL = prevSearch
	})

	meta := LRCLIBTrackMeta{Title: "T", Artist: "A", Album: "Al", DurationMs: 120_000}
	tr, err := FetchWithLRCLIBFallback(context.Background(), spotifySrv.Client(), "spotify:track:z", meta)
	if err != nil {
		t.Fatal(err)
	}
	if tr == nil || tr.SyncType != "LINE_SYNCED" || len(tr.Lines) != 2 {
		t.Fatalf("unexpected track: %+v", tr)
	}
	if tr.Lines[0].Words != "hello" || tr.Lines[0].StartMs != 1000 || tr.Lines[0].EndMs != 3500 {
		t.Fatalf("line0 = %+v", tr.Lines[0])
	}
	if tr.Lines[1].StartMs != 3500 || tr.Lines[1].EndMs != 0 {
		t.Fatalf("line1 end/start = %+v", tr.Lines[1])
	}
}
