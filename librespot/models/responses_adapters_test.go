package models

import (
	"strings"
	"testing"
)

func TestDecodePlayerEventMetadata(t *testing.T) {
	cover := "https://images.spotify.test/cover.jpg"
	raw := `{
		"type": "metadata",
		"data": {
			"context_uri": "spotify:album:album-id",
			"uri": "spotify:track:track-id",
			"name": "Track",
			"artist_names": ["Artist One", "Artist Two"],
			"album_name": "Album",
			"album_cover_url": "` + cover + `",
			"position": 42000,
			"duration": 180000
		}
	}`

	event, err := DecodePlayerEvent([]byte(raw))
	if err != nil {
		t.Fatalf("DecodePlayerEvent(metadata) error = %v, want nil", err)
	}
	if event.Type != EventTypeMetadata {
		t.Fatalf("event.Type = %q, want metadata", event.Type)
	}
	if event.Metadata == nil {
		t.Fatal("event.Metadata = nil, want payload")
	}
	if event.Playing != nil || event.Paused != nil || event.Stopped != nil || event.Seek != nil || event.Volume != nil {
		t.Fatalf("non-metadata payloads should be nil: %#v", event)
	}
	if event.Metadata.ContextURI != "spotify:album:album-id" || event.Metadata.URI != "spotify:track:track-id" {
		t.Fatalf("metadata URIs = %q/%q, want album context and track URI", event.Metadata.ContextURI, event.Metadata.URI)
	}
	if event.Metadata.Name != "Track" || event.Metadata.AlbumName != "Album" {
		t.Fatalf("metadata names = %q/%q, want Track/Album", event.Metadata.Name, event.Metadata.AlbumName)
	}
	if len(event.Metadata.ArtistNames) != 2 || event.Metadata.ArtistNames[0] != "Artist One" || event.Metadata.ArtistNames[1] != "Artist Two" {
		t.Fatalf("metadata artists = %#v, want two artist names", event.Metadata.ArtistNames)
	}
	if event.Metadata.AlbumCoverURL == nil || *event.Metadata.AlbumCoverURL != cover {
		t.Fatalf("metadata cover = %#v, want %q", event.Metadata.AlbumCoverURL, cover)
	}
	if event.Metadata.Position != 42000 || event.Metadata.Duration != 180000 {
		t.Fatalf("metadata position/duration = %d/%d, want 42000/180000", event.Metadata.Position, event.Metadata.Duration)
	}
}

func TestDecodePlayerEventTransportPayloads(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		assert func(t *testing.T, event PlayerEvent)
	}{
		{
			name: "playing",
			raw:  `{"type":"playing","data":{"context_uri":"spotify:playlist:p","uri":"spotify:track:t","resume":true,"play_origin":"lazyspotify"}}`,
			assert: func(t *testing.T, event PlayerEvent) {
				t.Helper()
				if event.Playing == nil {
					t.Fatal("event.Playing = nil, want payload")
				}
				if !event.Playing.Resume || event.Playing.URI != "spotify:track:t" || event.Playing.ContextURI != "spotify:playlist:p" {
					t.Fatalf("event.Playing = %#v, want resume payload for spotify:track:t", event.Playing)
				}
			},
		},
		{
			name: "paused",
			raw:  `{"type":"paused","data":{"context_uri":"spotify:playlist:p","uri":"spotify:track:t","play_origin":"lazyspotify"}}`,
			assert: func(t *testing.T, event PlayerEvent) {
				t.Helper()
				if event.Paused == nil {
					t.Fatal("event.Paused = nil, want payload")
				}
				if event.Paused.URI != "spotify:track:t" || event.Paused.ContextURI != "spotify:playlist:p" {
					t.Fatalf("event.Paused = %#v, want paused payload for spotify:track:t", event.Paused)
				}
			},
		},
		{
			name: "stopped",
			raw:  `{"type":"stopped","data":{"play_origin":"lazyspotify"}}`,
			assert: func(t *testing.T, event PlayerEvent) {
				t.Helper()
				if event.Stopped == nil {
					t.Fatal("event.Stopped = nil, want payload")
				}
				if event.Stopped.PlayOrigin != "lazyspotify" {
					t.Fatalf("event.Stopped.PlayOrigin = %q, want lazyspotify", event.Stopped.PlayOrigin)
				}
			},
		},
		{
			name: "seek",
			raw:  `{"type":"seek","data":{"context_uri":"spotify:playlist:p","uri":"spotify:track:t","position":5000,"duration":180000,"play_origin":"lazyspotify"}}`,
			assert: func(t *testing.T, event PlayerEvent) {
				t.Helper()
				if event.Seek == nil {
					t.Fatal("event.Seek = nil, want payload")
				}
				if event.Seek.Position != 5000 || event.Seek.Duration != 180000 {
					t.Fatalf("event.Seek position/duration = %d/%d, want 5000/180000", event.Seek.Position, event.Seek.Duration)
				}
			},
		},
		{
			name: "volume",
			raw:  `{"type":"volume","data":{"value":32768,"max":65535}}`,
			assert: func(t *testing.T, event PlayerEvent) {
				t.Helper()
				if event.Volume == nil {
					t.Fatal("event.Volume = nil, want payload")
				}
				if event.Volume.Value != 32768 || event.Volume.Max != 65535 {
					t.Fatalf("event.Volume = %#v, want 32768/65535", event.Volume)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := DecodePlayerEvent([]byte(tt.raw))
			if err != nil {
				t.Fatalf("DecodePlayerEvent(%s) error = %v, want nil", tt.name, err)
			}
			tt.assert(t, event)
		})
	}
}

func TestDecodePlayerEventRejectsUnsupportedEventType(t *testing.T) {
	_, err := DecodePlayerEvent([]byte(`{"type":"unknown","data":{}}`))
	if err == nil {
		t.Fatal("DecodePlayerEvent() error = nil, want unsupported event error")
	}
	if !strings.Contains(err.Error(), "unsupported event type: unknown") {
		t.Fatalf("DecodePlayerEvent() error = %q, want unsupported event error", err.Error())
	}
}

func TestDecodePlayerEventRejectsMalformedPayload(t *testing.T) {
	_, err := DecodePlayerEvent([]byte(`{"type":"volume","data":{"value":"loud","max":65535}}`))
	if err == nil {
		t.Fatal("DecodePlayerEvent() error = nil, want payload decode error")
	}
}
