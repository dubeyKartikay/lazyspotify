package lyrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ErrSpotifyForbidden is returned when Spotify's color-lyrics endpoint
// responds with HTTP 403 (common for third-party OAuth clients).
var ErrSpotifyForbidden = errors.New("lyrics: spotify color-lyrics forbidden")

// colorLyricsBase is the Spotify color-lyrics URL prefix (package tests may replace it).
var colorLyricsBase = "https://spclient.wg.spotify.com/color-lyrics/v2/track/"

// Line is one lyric line, optionally time-synced (StartMs / EndMs).
type Line struct {
	StartMs int64
	EndMs   int64
	Words   string
}

// Track holds parsed lyrics for a track.
type Track struct {
	SyncType string
	Lines    []Line
}

// TrackIDFromURI returns the Spotify track id for spotify:track:… URIs.
func TrackIDFromURI(uri string) (string, bool) {
	const prefix = "spotify:track:"
	if !strings.HasPrefix(uri, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(uri, prefix)
	if id == "" {
		return "", false
	}
	return id, true
}

// Fetch retrieves lyrics from Spotify's color-lyrics service using the same
// OAuth-backed HTTP client as the Web API (token refresh handled by oauth2).
func Fetch(ctx context.Context, httpClient *http.Client, trackURI string) (*Track, error) {
	trackID, ok := TrackIDFromURI(trackURI)
	if !ok {
		return nil, fmt.Errorf("lyrics: not a track uri: %s", trackURI)
	}
	if httpClient == nil {
		return nil, fmt.Errorf("lyrics: nil http client")
	}

	reqURL := colorLyricsBase + url.PathEscape(trackID) + "?format=json&vocalRemoval=false"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("app-platform", "WebPlayer")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return &Track{SyncType: "NONE", Lines: nil}, nil
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w (status 403)", ErrSpotifyForbidden)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("lyrics: unexpected status %d", resp.StatusCode)
	}

	var wire colorLyricsWire
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("lyrics: decode: %w", err)
	}
	if wire.Lyrics == nil {
		return &Track{SyncType: "NONE", Lines: nil}, nil
	}

	lines := make([]Line, 0, len(wire.Lyrics.Lines))
	for _, ln := range wire.Lyrics.Lines {
		start, _ := parseMs(ln.StartTimeMs)
		end, _ := parseMs(ln.EndTimeMs)
		lines = append(lines, Line{
			StartMs: start,
			EndMs:   end,
			Words:   strings.TrimSpace(ln.Words),
		})
	}
	return &Track{SyncType: wire.Lyrics.SyncType, Lines: lines}, nil
}

// CurrentLineIndex returns the index of the last line whose start is at or
// before positionMs, or -1 when nothing matches.
func CurrentLineIndex(lines []Line, positionMs int) int {
	if len(lines) == 0 || positionMs < 0 {
		return -1
	}
	pos := int64(positionMs)
	best := -1
	for i := range lines {
		if lines[i].StartMs <= pos {
			best = i
		} else {
			break
		}
	}
	return best
}

func parseMs(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

type colorLyricsWire struct {
	Lyrics *struct {
		SyncType string `json:"syncType"`
		Lines    []struct {
			StartTimeMs string `json:"startTimeMs"`
			EndTimeMs   string `json:"endTimeMs"`
			Words       string `json:"words"`
		} `json:"lines"`
	} `json:"lyrics"`
}
