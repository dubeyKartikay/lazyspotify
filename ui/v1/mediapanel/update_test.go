package mediapanel

import (
	"strings"
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

func TestMoreInfoOpensForSelectedItem(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())
	model.SetSize(30, 12)
	setActivePanelContent(t, &model, []common.Entity{
		common.NewEntity("An intentionally long playlist name", "A full playlist description that should stay intact inside the info strip.", "playlist:1", ""),
	})

	cmd := model.Update(textKey("i"))
	if cmd != nil {
		t.Fatalf("open info returned unexpected cmd")
	}
	if !model.infoOpen {
		t.Fatal("expected info mode to be open")
	}
	if !strings.Contains(model.infoViewport.GetContent(), "An intentionally long playlist name") {
		t.Fatal("expected full name in info viewport content")
	}
	if !strings.Contains(model.infoViewport.GetContent(), "A full playlist description that should stay intact inside the info strip.") {
		t.Fatal("expected full description in info viewport content")
	}
}

func TestMoreInfoCloseKeys(t *testing.T) {
	closeKeys := []tea.KeyPressMsg{
		textKey("i"),
		keyCode(tea.KeyEsc),
		keyCode(tea.KeyBackspace),
	}

	for _, closeKey := range closeKeys {
		model := NewModel(common.NewAppKeyMap())
		model.SetSize(30, 12)
		setActivePanelContent(t, &model, []common.Entity{
			common.NewEntity("Playlist", "Description", "playlist:1", ""),
		})

		model.Update(textKey("i"))
		if !model.infoOpen {
			t.Fatal("expected info mode to be open")
		}

		cmd := model.Update(closeKey)
		if cmd != nil {
			t.Fatalf("close key %q returned unexpected cmd", closeKey.String())
		}
		if model.infoOpen {
			t.Fatalf("expected info mode to close for %q", closeKey.String())
		}
	}
}

func TestBackClosesInfoBeforePoppingNestedList(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())
	model.SetSize(30, 12)
	model.panels[0].lists.Push(medialist.NewModel(common.Tracks))
	setActivePanelContent(t, &model, []common.Entity{
		common.NewEntity("Track", strings.Repeat("Nested track description ", 20), "track:1", ""),
	})

	model.Update(textKey("i"))
	if !model.infoOpen {
		t.Fatal("expected info mode to be open")
	}

	cmd := model.Update(keyCode(tea.KeyBackspace))
	if cmd != nil {
		t.Fatal("expected closing info to not return a command")
	}
	if model.infoOpen {
		t.Fatal("expected info mode to close")
	}
	if model.activePanel().depth() != 2 {
		t.Fatalf("depth = %d, want 2", model.activePanel().depth())
	}
}

func TestMoreInfoFollowsSelectionAndResetsScroll(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())
	model.SetSize(20, 12)
	setActivePanelContent(t, &model, []common.Entity{
		common.NewEntity("First track", strings.Repeat("First description ", 30), "track:1", ""),
		common.NewEntity("Second track", strings.Repeat("Second description ", 30), "track:2", ""),
	})

	model.Update(textKey("i"))
	model.Update(ctrlKey('d'))
	if model.infoViewport.YOffset() == 0 {
		t.Fatal("expected viewport to scroll down")
	}

	model.Update(keyCode(tea.KeyDown))

	if got := model.infoViewport.YOffset(); got != 0 {
		t.Fatalf("viewport y-offset = %d, want 0", got)
	}
	if !strings.Contains(model.infoViewport.GetContent(), "Second track") {
		t.Fatal("expected info strip to follow the selected item")
	}
}

func TestMoreInfoCtrlScrollViewport(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())
	model.SetSize(20, 12)
	setActivePanelContent(t, &model, []common.Entity{
		common.NewEntity("Track", strings.Repeat("Long description ", 40), "track:1", ""),
	})

	model.Update(textKey("i"))
	model.Update(ctrlKey('d'))
	downOffset := model.infoViewport.YOffset()
	if downOffset == 0 {
		t.Fatal("expected scroll down to move the viewport")
	}

	model.Update(ctrlKey('u'))
	if got := model.infoViewport.YOffset(); got >= downOffset {
		t.Fatalf("viewport y-offset = %d, want less than %d after scrolling up", got, downOffset)
	}
}

func TestBracketsPaginateAndCtrlScrollsInfo(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())
	model.SetSize(20, 12)
	setActivePanelContentWithPagination(t, &model, []common.Entity{
		common.NewEntity("Track", strings.Repeat("Long description ", 40), "track:1", ""),
	}, common.PaginationInfo{
		CurrentPage: 1,
		TotalPages:  2,
		TotalItems:  20,
		HasNext:     true,
		NextCursor:  "cursor:2",
	}, common.MediaRequest{
		Kind:        common.GetUserPlaylists,
		PanelKind:   common.Playlists,
		Page:        1,
		ShowLoading: true,
	})

	cmd := model.Update(textKey("]"))
	request, ok := cmd().(common.MediaRequest)
	if !ok {
		t.Fatalf("closed info ] returned %T, want common.MediaRequest", cmd())
	}
	if request.Cursor != "cursor:2" {
		t.Fatalf("cursor = %q, want cursor:2", request.Cursor)
	}

	model.Update(textKey("i"))
	beforeOffset := model.infoViewport.YOffset()
	cmd = model.Update(textKey("]"))
	requestMsg, ok := cmd().(common.MediaRequest)
	if !ok {
		t.Fatalf("info-open ] returned %T, want common.MediaRequest", cmd())
	}
	if requestMsg.Cursor != "cursor:2" {
		t.Fatalf("cursor = %q, want cursor:2", requestMsg.Cursor)
	}

	cmd = model.Update(ctrlKey('d'))
	if cmd != nil {
		t.Fatal("expected ctrl+d to scroll info, not paginate, when info mode is open")
	}
	if model.infoViewport.YOffset() <= beforeOffset {
		t.Fatal("expected ctrl+d to scroll info mode down")
	}

	cmd = model.Update(textKey("l"))
	requestMsg, ok = cmd().(common.MediaRequest)
	if !ok {
		t.Fatalf("info-open l returned %T, want common.MediaRequest", cmd())
	}
	if requestMsg.Cursor != "cursor:2" {
		t.Fatalf("cursor = %q, want cursor:2", requestMsg.Cursor)
	}
}

func TestSearchFocusKeepsLiteralIForInput(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())
	model.SetSize(30, 12)

	model.focusSearch()
	model.Update(textKey("i"))

	if model.infoOpen {
		t.Fatal("expected info mode to remain closed while search is focused")
	}
	if got := model.searchInput.Value(); got != "i" {
		t.Fatalf("search input = %q, want i", got)
	}
}

func TestMoreInfoWithoutSelectionShowsStatus(t *testing.T) {
	model := NewModel(common.NewAppKeyMap())
	model.SetSize(30, 12)

	cmd := model.Update(textKey("i"))
	if cmd == nil {
		t.Fatal("expected status command when no item is selected")
	}
	if model.infoOpen {
		t.Fatal("expected info mode to stay closed")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("expected status command to emit a message")
	}
}

func setActivePanelContent(t *testing.T, model *Model, entities []common.Entity) {
	t.Helper()
	setActivePanelContentWithPagination(t, model, entities, common.PaginationInfo{
		CurrentPage: 1,
		TotalPages:  1,
		TotalItems:  len(entities),
	}, requestForActiveList(model))
}

func setActivePanelContentWithPagination(t *testing.T, model *Model, entities []common.Entity, pagination common.PaginationInfo, request common.MediaRequest) {
	t.Helper()
	cmd := model.SetContent(entities, model.activePanel().activeList().Kind(), pagination, request)
	if cmd != nil {
		msg := cmd()
		if msg == nil {
			t.Fatal("expected set content command to emit a message")
		}
		model.activePanel().activeList().Update(msg)
	}
}

func textKey(text string) tea.KeyPressMsg {
	runes := []rune(text)
	return tea.KeyPressMsg(tea.Key{Text: text, Code: runes[0]})
}

func keyCode(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code})
}

func ctrlKey(char rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: char, Mod: tea.ModCtrl})
}

func requestForActiveList(model *Model) common.MediaRequest {
	request := common.MediaRequest{
		PanelKind:   model.activePanel().kind,
		Page:        1,
		ShowLoading: true,
	}

	switch model.activePanel().activeList().Kind() {
	case common.Tracks:
		switch model.activePanel().kind {
		case common.Albums:
			request.Kind = common.GetAlbumTracks
		default:
			request.Kind = common.GetPlaylistTracks
		}
	case common.Albums:
		request.Kind = common.GetSavedAlbums
	case common.Artists:
		request.Kind = common.GetFollowedArtists
	default:
		request.Kind = common.GetUserPlaylists
	}

	return request
}
