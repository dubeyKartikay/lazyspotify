package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
	spotifyapi "github.com/zmb3/spotify/v2"
)

func TestCursorEncodingAndDecodingAreDefensive(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
		want   int
	}{
		{name: "empty", cursor: "", want: 0},
		{name: "valid", cursor: "20", want: 20},
		{name: "negative", cursor: "-10", want: 0},
		{name: "invalid", cursor: "not-a-number", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeOffsetCursor(tt.cursor); got != tt.want {
				t.Fatalf("decodeOffsetCursor(%q) = %d, want %d", tt.cursor, got, tt.want)
			}
		})
	}

	if got := encodeOffsetCursor(-20); got != "0" {
		t.Fatalf("encodeOffsetCursor(-20) = %q, want 0", got)
	}
	if got := encodeOffsetCursor(30); got != "30" {
		t.Fatalf("encodeOffsetCursor(30) = %q, want 30", got)
	}
}

func TestPaginationFromOffsetReflectsCurrentPageAndNextCursor(t *testing.T) {
	got := paginationFromOffset(10, 10, 25, 10)
	want := common.PaginationInfo{
		CurrentPage: 2,
		TotalPages:  3,
		TotalItems:  25,
		HasNext:     true,
		NextCursor:  "20",
	}
	if got != want {
		t.Fatalf("paginationFromOffset() = %#v, want %#v", got, want)
	}

	got = paginationFromOffset(20, 5, 25, 10)
	want = common.PaginationInfo{
		CurrentPage: 3,
		TotalPages:  3,
		TotalItems:  25,
		HasNext:     false,
		NextCursor:  "",
	}
	if got != want {
		t.Fatalf("paginationFromOffset(last page) = %#v, want %#v", got, want)
	}
}

func TestHandleMediaRequestDefaultsPageBeforeDispatch(t *testing.T) {
	model := NewModel()

	var gotRequest common.MediaRequest
	model.requestHandlers[common.GetSavedTracks] = func(request common.MediaRequest) tea.Cmd {
		gotRequest = request
		return func() tea.Msg {
			return "handled"
		}
	}

	cmd := model.handleMediaRequest(common.MediaRequest{Kind: common.GetSavedTracks})
	if cmd == nil {
		t.Fatal("handleMediaRequest() = nil, want dispatch command")
	}
	if msg := cmd(); msg != "handled" {
		t.Fatalf("handler message = %#v, want handled", msg)
	}
	if gotRequest.Page != 1 {
		t.Fatalf("dispatched Page = %d, want 1", gotRequest.Page)
	}
}

func TestHandleMediaRequestReturnsNilForUnknownKind(t *testing.T) {
	model := NewModel()

	cmd := model.handleMediaRequest(common.MediaRequest{Kind: common.MediaRequestKind(999), Page: 1})
	if cmd != nil {
		t.Fatalf("handleMediaRequest(unknown) = %#v, want nil", cmd)
	}
}

func TestAdaptSpotifySavedTracksBuildsUserFacingEntity(t *testing.T) {
	page := &spotifyapi.SavedTrackPage{
		Tracks: []spotifyapi.SavedTrack{
			{
				FullTrack: spotifyapi.FullTrack{
					SimpleTrack: spotifyapi.SimpleTrack{
						Name: "Track",
						URI:  "spotify:track:track-id",
						Artists: []spotifyapi.SimpleArtist{
							{Name: "Artist One"},
							{Name: "Artist Two"},
						},
					},
					Album: spotifyapi.SimpleAlbum{
						Name:   "Album",
						Images: []spotifyapi.Image{{URL: "https://images.spotify.test/large.jpg"}, {URL: "https://images.spotify.test/small.jpg"}},
					},
				},
			},
		},
	}

	entities := adaptSpotifySavedTracks(page)
	if len(entities) != 1 {
		t.Fatalf("len(entities) = %d, want 1", len(entities))
	}

	want := common.NewEntity("Track", "Artist One, Artist Two • Album", "spotify:track:track-id", "https://images.spotify.test/large.jpg")
	if entities[0] != want {
		t.Fatalf("entity = %#v, want %#v", entities[0], want)
	}
}

func TestAdaptResolvedPlaylistTracksBuildsDescriptionsWithoutDanglingSeparators(t *testing.T) {
	tracks := []models.ResolvedTrack{
		{
			Name:      "Track With Artists And Album",
			URI:       "spotify:track:both",
			Artists:   []string{"Artist One", "Artist Two"},
			AlbumName: "Album",
			Img:       "https://images.spotify.test/track.jpg",
		},
		{
			Name:      "Album Only Track",
			URI:       "spotify:track:album-only",
			AlbumName: "Album Only",
		},
		{
			Name:    "Artists Only Track",
			URI:     "spotify:track:artists-only",
			Artists: []string{"Artist"},
		},
	}

	entities := adaptResolvedPlaylistTracks(tracks)
	if len(entities) != 3 {
		t.Fatalf("len(entities) = %d, want 3", len(entities))
	}
	assertEntity(t, entities[0], common.NewEntity("Track With Artists And Album", "Artist One, Artist Two • Album", "spotify:track:both", "https://images.spotify.test/track.jpg"))
	assertEntity(t, entities[1], common.NewEntity("Album Only Track", "Album Only", "spotify:track:album-only", ""))
	assertEntity(t, entities[2], common.NewEntity("Artists Only Track", "Artist", "spotify:track:artists-only", ""))
}

func TestApplyPlayerEventUpdatesPlaybackState(t *testing.T) {
	model := NewModel()
	model.playing = true
	model.volumeInfo = common.VolumeInfo{Volume: 1000, Max: 65535}

	model.applyPlayerEvent(models.PlayerEvent{
		Type: models.EventTypeMetadata,
		Metadata: &models.MetadataEventData{
			Name:        "Track",
			ArtistNames: []string{"Artist One", "Artist Two"},
			AlbumName:   "Album",
			Position:    42000,
			Duration:    180000,
		},
	})
	wantSong := common.SongInfo{
		Title:    "Track",
		Artist:   "Artist One, Artist Two",
		Album:    "Album",
		Position: 42000,
		Duration: 180000,
	}
	if model.songInfo != wantSong {
		t.Fatalf("songInfo = %#v, want %#v", model.songInfo, wantSong)
	}

	model.applyPlayerEvent(models.PlayerEvent{
		Type: models.EventTypeSeek,
		Seek: &models.SeekEventData{Position: 60000, Duration: 181000},
	})
	if model.songInfo.Position != 60000 || model.songInfo.Duration != 181000 {
		t.Fatalf("seek-updated songInfo = %#v, want position 60000 duration 181000", model.songInfo)
	}

	model.applyPlayerEvent(models.PlayerEvent{
		Type:   models.EventTypeVolume,
		Volume: &models.VolumeEventData{Value: 2000, Max: 0},
	})
	if model.volumeInfo.Volume != 2000 || model.volumeInfo.Max != 65535 {
		t.Fatalf("volumeInfo after max=0 event = %#v, want value 2000 and previous max 65535", model.volumeInfo)
	}

	model.applyPlayerEvent(models.PlayerEvent{
		Type:   models.EventTypeVolume,
		Volume: &models.VolumeEventData{Value: 50, Max: 100},
	})
	if model.volumeInfo.Volume != 50 || model.volumeInfo.Max != 100 {
		t.Fatalf("volumeInfo after max=100 event = %#v, want 50/100", model.volumeInfo)
	}

	model.applyPlayerEvent(models.PlayerEvent{Type: models.EventTypeStopped})
	if model.playing {
		t.Fatal("playing = true after stopped event, want false")
	}
	if model.songInfo.Position != 0 {
		t.Fatalf("songInfo.Position after stopped event = %d, want 0", model.songInfo.Position)
	}
}

func assertEntity(t *testing.T, got common.Entity, want common.Entity) {
	t.Helper()
	if got != want {
		t.Fatalf("entity = %#v, want %#v", got, want)
	}
}
