package lyrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// lrclibGetURL and lrclibSearchURL may be replaced in package tests.
var (
	lrclibGetURL    = "https://lrclib.net/api/get"
	lrclibSearchURL = "https://lrclib.net/api/search"
)

// LRCLIBTrackMeta identifies a track for LRCLIB's /api/get lookup.
// DurationMs should be the track length in milliseconds (e.g. from playback metadata).
type LRCLIBTrackMeta struct {
	Title      string
	Artist     string
	Album      string
	DurationMs int
}

var lrcLineRe = regexp.MustCompile(`^\[(\d+):(\d+)(?:\.(\d+))?\]\s*(.*)$`)

// FetchWithLRCLIBFallback tries Spotify color-lyrics first; if Spotify returns 403,
// it fetches time-synced lyrics from LRCLIB (https://lrclib.net).
func FetchWithLRCLIBFallback(ctx context.Context, spotifyClient *http.Client, trackURI string, meta LRCLIBTrackMeta) (*Track, error) {
	tr, err := Fetch(ctx, spotifyClient, trackURI)
	if err == nil {
		return tr, nil
	}
	if !errors.Is(err, ErrSpotifyForbidden) {
		return nil, err
	}
	tr, errLR := FetchLRCLIB(ctx, nil, meta)
	if errLR != nil {
		return nil, fmt.Errorf("%w; lrclib: %v", ErrSpotifyForbidden, errLR)
	}
	return tr, nil
}

// FetchLRCLIB loads synced lyrics from LRCLIB's public API. httpClient may be nil
// (a sensible default timeout is used). meta.Title and meta.Artist should be non-empty
// for reliable matches.
func FetchLRCLIB(ctx context.Context, httpClient *http.Client, meta LRCLIBTrackMeta) (*Track, error) {
	if strings.TrimSpace(meta.Title) == "" && strings.TrimSpace(meta.Artist) == "" {
		return nil, fmt.Errorf("lrclib: need title or artist")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}

	if meta.DurationMs > 0 {
		q := lrclibQueryValues(meta)
		durSec := (meta.DurationMs + 500) / 1000
		if durSec < 1 {
			durSec = 1
		}
		q.Set("duration", strconv.Itoa(durSec))
		tr, err := lrclibDoGet(ctx, httpClient, lrclibGetURL+"?"+q.Encode())
		if err != nil {
			return nil, err
		}
		if tr != nil && len(tr.Lines) > 0 {
			return tr, nil
		}
	}
	return lrclibSearch(ctx, httpClient, meta)
}

func lrclibQueryValues(meta LRCLIBTrackMeta) url.Values {
	q := url.Values{}
	q.Set("track_name", strings.TrimSpace(meta.Title))
	q.Set("artist_name", strings.TrimSpace(meta.Artist))
	q.Set("album_name", strings.TrimSpace(meta.Album))
	return q
}

// lrclibDoGet requests a single LRCLIB record. On HTTP 404 it returns nil, nil.
// On HTTP 200 with a usable record it returns a non-nil *Track (possibly zero lines
// when instrumental).
func lrclibDoGet(ctx context.Context, httpClient *http.Client, reqURL string) (*Track, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	setLRCLIBRequestHeaders(req)

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
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("lrclib: http %d", resp.StatusCode)
	}

	var wire lrclibGetWire
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("lrclib: decode: %w", err)
	}
	if wire.Instrumental {
		return &Track{SyncType: "NONE", Lines: nil}, nil
	}
	tr, err := trackFromLRCLIBRecord(&wire)
	if err != nil {
		return nil, err
	}
	if tr != nil && len(tr.Lines) > 0 {
		return tr, nil
	}
	return nil, nil
}

func lrclibSearch(ctx context.Context, httpClient *http.Client, meta LRCLIBTrackMeta) (*Track, error) {
	q := lrclibQueryValues(meta)
	reqURL := lrclibSearchURL + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	setLRCLIBRequestHeaders(req)

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
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("lrclib: search http %d", resp.StatusCode)
	}

	var records []lrclibGetWire
	if err := json.Unmarshal(body, &records); err != nil {
		return nil, fmt.Errorf("lrclib: search decode: %w", err)
	}
	for i := range records {
		if records[i].Instrumental {
			continue
		}
		tr, err := trackFromLRCLIBRecord(&records[i])
		if err != nil {
			return nil, err
		}
		if tr != nil && len(tr.Lines) > 0 {
			return tr, nil
		}
	}
	return &Track{SyncType: "NONE", Lines: nil}, nil
}

func trackFromLRCLIBRecord(wire *lrclibGetWire) (*Track, error) {
	if wire == nil {
		return nil, nil
	}
	raw := ""
	if wire.SyncedLyrics != nil {
		raw = strings.TrimSpace(*wire.SyncedLyrics)
	}
	if raw == "" {
		return nil, nil
	}
	lines, err := parseLRC(raw)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, nil
	}
	fillLineEndMs(lines)
	return &Track{SyncType: "LINE_SYNCED", Lines: lines}, nil
}

type lrclibGetWire struct {
	SyncedLyrics *string `json:"syncedLyrics"`
	Instrumental bool    `json:"instrumental"`
}

func parseLRC(raw string) ([]Line, error) {
	var lines []Line
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := lrcLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		startMs := lrcTimestampToMs(m[1], m[2], m[3])
		words := strings.TrimSpace(m[4])
		if words == "" {
			continue
		}
		lines = append(lines, Line{StartMs: startMs, EndMs: 0, Words: words})
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("lrclib: no timed lines in synced lyrics")
	}
	return lines, nil
}

func lrcTimestampToMs(minStr, secStr, fracStr string) int64 {
	min, _ := strconv.ParseInt(minStr, 10, 64)
	sec, _ := strconv.ParseInt(secStr, 10, 64)
	base := min*60*1000 + sec*1000
	fracStr = strings.TrimSpace(fracStr)
	if fracStr == "" {
		return base
	}
	n, _ := strconv.ParseInt(fracStr, 10, 64)
	switch len(fracStr) {
	case 1:
		return base + n*100
	case 2:
		return base + n*10
	default:
		return base + n
	}
}

func fillLineEndMs(lines []Line) {
	for i := range lines {
		if i+1 < len(lines) {
			lines[i].EndMs = lines[i+1].StartMs
		}
	}
}

func setLRCLIBRequestHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "lazyspotify (https://github.com/dubeyKartikay/lazyspotify)")
	req.Header.Set("Lrclib-Client", "https://github.com/dubeyKartikay/lazyspotify")
}
