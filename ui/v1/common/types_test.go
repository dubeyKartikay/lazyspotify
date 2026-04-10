package common

import "testing"

func TestRootMediaRequestForListKind(t *testing.T) {
	tests := []struct {
		name      string
		kind      ListKind
		query     string
		wantKind  MediaRequestKind
		wantQuery string
	}{
		{name: "playlists library", kind: Playlists, wantKind: GetUserPlaylists},
		{name: "tracks library", kind: Tracks, wantKind: GetSavedTracks},
		{name: "albums library", kind: Albums, wantKind: GetSavedAlbums},
		{name: "artists library", kind: Artists, wantKind: GetFollowedArtists},
		{name: "playlists search", kind: Playlists, query: " green ", wantKind: SearchPlaylists, wantQuery: "green"},
		{name: "tracks search", kind: Tracks, query: "green", wantKind: SearchTracks, wantQuery: "green"},
		{name: "albums search", kind: Albums, query: "green", wantKind: SearchAlbums, wantQuery: "green"},
		{name: "artists search", kind: Artists, query: "green", wantKind: SearchArtists, wantQuery: "green"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := RootMediaRequestForListKind(tt.kind, tt.query)
			if request.Kind != tt.wantKind {
				t.Fatalf("kind = %v, want %v", request.Kind, tt.wantKind)
			}
			if request.Query != tt.wantQuery {
				t.Fatalf("query = %q, want %q", request.Query, tt.wantQuery)
			}
			if request.Page != 1 {
				t.Fatalf("page = %d, want 1", request.Page)
			}
			if !request.ShowLoading {
				t.Fatal("request should enable loading")
			}
		})
	}
}
