package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	uiauth "github.com/dubeyKartikay/lazyspotify/ui/v1/auth"
)

func TestCallbackInputReceivesURLPunctuation(t *testing.T) {
	m := NewModel()
	m.authModel.SetState(uiauth.Authenticating)
	m.authModel.Authenticator().AuthServer.Started.Store(true)
	m.Update(tea.KeyPressMsg(tea.Key{Text: "p", Code: 'p'}))
	if !m.authModel.EnteringCallback() {
		t.Fatal("paste mode did not open")
	}
	if _, handled := m.handleShellInput(tea.KeyPressMsg(tea.Key{Text: "?", Code: '?'})); handled {
		t.Fatal("help shortcut swallowed callback URL punctuation")
	}
	if _, handled := m.handleShellInput(tea.KeyPressMsg(tea.Key{Code: 'c', Mod: tea.ModCtrl})); !handled {
		t.Fatal("Ctrl+C should still quit during callback entry")
	}
}
