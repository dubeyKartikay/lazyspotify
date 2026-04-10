package medialist

import (
	"testing"

	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
)

func TestPaginationRequestsPreserveSearchQuery(t *testing.T) {
	model := NewModel(common.Playlists)

	model.ApplyPagination(common.PaginationInfo{
		CurrentPage: 1,
		TotalPages:  3,
		TotalItems:  30,
		HasNext:     true,
		NextCursor:  "10",
	}, common.MediaRequest{
		Kind:        common.SearchPlaylists,
		Page:        1,
		Query:       "green",
		ShowLoading: true,
	})

	next, ok := model.NextPageRequest()
	if !ok {
		t.Fatal("expected next page request")
	}
	if next.Query != "green" {
		t.Fatalf("next query = %q, want green", next.Query)
	}
	if next.Kind != common.SearchPlaylists {
		t.Fatalf("next kind = %v, want %v", next.Kind, common.SearchPlaylists)
	}
	if next.Cursor != "10" {
		t.Fatalf("next cursor = %q, want 10", next.Cursor)
	}

	model.ApplyPagination(common.PaginationInfo{
		CurrentPage: 2,
		TotalPages:  3,
		TotalItems:  30,
		HasNext:     true,
		NextCursor:  "20",
	}, common.MediaRequest{
		Kind:        common.SearchPlaylists,
		Cursor:      "10",
		Page:        2,
		Query:       "green",
		ShowLoading: true,
	})

	prev, ok := model.PrevPageRequest()
	if !ok {
		t.Fatal("expected previous page request")
	}
	if prev.Query != "green" {
		t.Fatalf("prev query = %q, want green", prev.Query)
	}
	if prev.Kind != common.SearchPlaylists {
		t.Fatalf("prev kind = %v, want %v", prev.Kind, common.SearchPlaylists)
	}
	if prev.Cursor != "" {
		t.Fatalf("prev cursor = %q, want empty", prev.Cursor)
	}
}
