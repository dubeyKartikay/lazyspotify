package mediapanel

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/medialist"
)

func TestApplySearchResetsPanelsToRoot(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())
	model.panels[0].lists.Push(medialist.NewModel(common.Playlists))
	model.panels[1].lists.Push(medialist.NewModel(common.Tracks))

	cmd := model.applySearch("green")
	if model.searchQuery != "green" {
		t.Fatalf("searchQuery = %q, want green", model.searchQuery)
	}

	for i, panel := range model.panels {
		if panel.depth() != 1 {
			t.Fatalf("panel %d depth = %d, want 1", i, panel.depth())
		}
		if panel.activeList().State() != medialist.Initialized {
			t.Fatalf("panel %d state = %v, want initialized", i, panel.activeList().State())
		}
	}

	msg := cmd()
	request, ok := msg.(common.MediaRequest)
	if !ok {
		t.Fatalf("cmd returned %T, want common.MediaRequest", msg)
	}
	if request.Kind != common.SearchPlaylists {
		t.Fatalf("kind = %v, want %v", request.Kind, common.SearchPlaylists)
	}
	if request.Query != "green" {
		t.Fatalf("query = %q, want green", request.Query)
	}
}

func TestBackAtRootClearsSearchAndReloadsLibrary(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())
	model.searchQuery = "green"
	model.searchInput.SetValue("green")

	cmd := model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))
	if cmd == nil {
		t.Fatal("expected clear-search command")
	}
	if model.searchQuery != "" {
		t.Fatalf("searchQuery = %q, want empty", model.searchQuery)
	}
	if model.activePanel().depth() != 1 {
		t.Fatalf("depth = %d, want 1", model.activePanel().depth())
	}

	msg := cmd()
	request, ok := msg.(common.MediaRequest)
	if !ok {
		t.Fatalf("cmd returned %T, want common.MediaRequest", msg)
	}
	if request.Kind != common.GetUserPlaylists {
		t.Fatalf("kind = %v, want %v", request.Kind, common.GetUserPlaylists)
	}
	if request.Query != "" {
		t.Fatalf("query = %q, want empty", request.Query)
	}
}

func TestBackFromNestedListDoesNotClearSearch(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())
	model.searchQuery = "green"
	model.panels[0].lists.Push(medialist.NewModel(common.Playlists))

	model.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyBackspace}))

	if model.searchQuery != "green" {
		t.Fatalf("searchQuery = %q, want green", model.searchQuery)
	}
	if model.activePanel().depth() != 1 {
		t.Fatalf("depth = %d, want 1 after pop", model.activePanel().depth())
	}
}
