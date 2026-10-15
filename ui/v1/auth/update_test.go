package auth

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestRemoteCallbackPasteAndRetry(t *testing.T) {
	m := NewModel()
	m.SetState(Authenticating)
	m.SetSize(120, 35)
	m.auth.AuthServer.Started.Store(true)
	m.Update(tea.KeyPressMsg(tea.Key{Text: "p", Code: 'p'}))
	if !m.EnteringCallback() {
		t.Fatal("p did not focus the callback input")
	}
	callback := "http://127.0.0.1:8287/callback?code=auth-code&state=session-state"
	m.Update(tea.PasteMsg{Content: callback})
	if m.pasteInput.Value() != callback {
		t.Fatal("terminal paste did not preserve the full callback URL")
	}
	_, submit := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if submit == nil || !m.pasteSubmitting {
		t.Fatal("Enter did not start the callback exchange")
	}
	_, duplicate := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if duplicate != nil {
		t.Fatal("repeated Enter started a duplicate exchange")
	}
	m.Update(tea.PasteMsg{Content: "another callback"})
	if m.pasteInput.Value() != callback {
		t.Fatal("callback input changed while an exchange was pending")
	}
	m.Update(pasteResultMsg{err: errors.New("invalid callback")})
	if !m.pasteFocused || m.pasteSubmitting || !strings.Contains(m.View().Content, "invalid callback") {
		t.Fatal("failed exchange should show an error and allow retrying")
	}
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if m.pasteFocused || m.pasteErr != nil {
		t.Fatal("Escape did not dismiss the callback input and error")
	}
}

func TestSubmissionCapturesInputAndKeepsAuthFlowAliveOnInvalidURL(t *testing.T) {
	m := NewModel()
	m.SetState(Authenticating)
	m.focusPaste()
	m.pasteInput.SetValue("not-a-callback-url")
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	// Commands execute asynchronously; subsequent edits must not alter the value
	// that was submitted. This input fails validation without network access.
	m.pasteInput.SetValue("")
	result := cmd().(pasteResultMsg)
	if result.err == nil || !strings.Contains(result.err.Error(), "full address bar URL") {
		t.Fatalf("unexpected exchange result: %v", result.err)
	}
	m.Update(result)
	if m.err != nil || m.State() != Authenticating || !m.EnteringCallback() {
		t.Fatal("invalid paste should not terminate the browser login")
	}
}
