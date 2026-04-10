package medialist

import (
	"testing"

	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
)

func TestNewModelDisablesListQuitKeybindings(t *testing.T) {
	model := NewModel(common.Playlists)

	if model.list.KeyMap.Quit.Enabled() {
		t.Fatal("expected list quit keybinding to be disabled")
	}
	if model.list.KeyMap.ForceQuit.Enabled() {
		t.Fatal("expected list force-quit keybinding to be disabled")
	}
}
