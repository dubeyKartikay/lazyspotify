package lyrics

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTrackIDFromURI(t *testing.T) {
	id, ok := TrackIDFromURI("spotify:track:abcXYZ012")
	if !ok || id != "abcXYZ012" {
		t.Fatalf("TrackIDFromURI = %q, %v want abcXYZ012, true", id, ok)
	}
	_, ok = TrackIDFromURI("spotify:episode:abc")
	if ok {
		t.Fatal("TrackIDFromURI(episode) = true, want false")
	}
}

func TestCurrentLineIndex(t *testing.T) {
	lines := []Line{
		{StartMs: 0, EndMs: 2000, Words: "a"},
		{StartMs: 2000, EndMs: 5000, Words: "b"},
		{StartMs: 5000, EndMs: 0, Words: "c"},
	}
	if got := CurrentLineIndex(lines, 0); got != 0 {
		t.Fatalf("idx at 0 = %d want 0", got)
	}
	if got := CurrentLineIndex(lines, 2500); got != 1 {
		t.Fatalf("idx at 2500 = %d want 1", got)
	}
	if got := CurrentLineIndex(lines, 6000); got != 2 {
		t.Fatalf("idx at 6000 = %d want 2", got)
	}
	if got := CurrentLineIndex(nil, 100); got != -1 {
		t.Fatalf("idx empty = %d want -1", got)
	}
}

func TestDecodeSampleColorLyricsJSON(t *testing.T) {
	raw := `{
  "lyrics": {
    "syncType": "LINE_SYNCED",
    "lines": [
      {"startTimeMs": "0", "endTimeMs": "1500", "words": "hello"},
      {"startTimeMs": "1500", "endTimeMs": "3000", "words": "world"}
    ]
  }
}`
	var wire colorLyricsWire
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Lyrics == nil || wire.Lyrics.SyncType != "LINE_SYNCED" {
		t.Fatalf("syncType = %v", wire.Lyrics)
	}
	if len(wire.Lyrics.Lines) != 2 {
		t.Fatalf("lines len = %d", len(wire.Lyrics.Lines))
	}
}

func TestParseLRC(t *testing.T) {
	raw := "[00:00.50]first\n[01:02.03]second line\n[ar:ignore]\n"
	lines, err := parseLRC(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("len = %d", len(lines))
	}
	if lines[0].StartMs != 500 || lines[0].Words != "first" {
		t.Fatalf("line0 = %+v", lines[0])
	}
	want := int64(1*60*1000 + 2*1000 + 30) // [01:02.03] → 62030 ms
	if lines[1].StartMs != want || !strings.Contains(lines[1].Words, "second") {
		t.Fatalf("line1 = %+v", lines[1])
	}
}
