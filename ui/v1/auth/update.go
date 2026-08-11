package auth

import (
	"context"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	coreauth "github.com/dubeyKartikay/lazyspotify/core/auth"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
)

type authQuitMsg struct{}

type pasteResultMsg struct{ err error }

var keyMap = struct {
	CopyURL  key.Binding
	PasteURL key.Binding
}{
	CopyURL:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy url")),
	PasteURL: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "paste callback")),
}

func (m *Model) startAuthFlow() tea.Msg {
	_, err := m.auth.ReAuthenticate(context.Background(), m.authFlowUpdates)
	if err != nil {
		return coreauth.AuthServerErr{Err: err}
	}
	return nil
}

func (m *Model) listenForAuthUpdates() tea.Msg {
	updates := <-m.authFlowUpdates
	if updates == "success" {
		return Authenticated
	}
	return Authenticating
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) quitAfterError() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		return authQuitMsg{}
	}
}

func (m *Model) submitPastedCallback() tea.Cmd {
	pasted := m.pasteInput.Value()
	authenticator := m.auth
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := authenticator.CompleteLoginWithPastedCallback(ctx, pasted)
		return pasteResultMsg{err: err}
	}
}

func (m *Model) focusPaste() tea.Cmd {
	m.pasteFocused = true
	m.pasteErr = nil
	m.pasteInput.SetValue("")
	m.pasteInput.CursorEnd()
	return m.pasteInput.Focus()
}

func (m *Model) blurPaste() {
	m.pasteFocused = false
	m.pasteErr = nil
	m.pasteInput.Blur()
}

func (m *Model) updatePasteInput(msg tea.KeyPressMsg) tea.Cmd {
	if m.pasteSubmitting {
		return nil
	}
	switch msg.String() {
	case "enter":
		m.pasteErr = nil
		m.pasteSubmitting = true
		return m.submitPastedCallback()
	case "esc":
		m.blurPaste()
		return nil
	}
	var cmd tea.Cmd
	m.pasteInput, cmd = m.pasteInput.Update(msg)
	return cmd
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.authState == NeedsAuth {
		m.authState = Authenticating
		return m, tea.Batch(m.startAuthFlow, m.listenForAuthUpdates)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.pasteFocused {
			return m, m.updatePasteInput(msg)
		}
		if key.Matches(msg, keyMap.CopyURL) && m.auth.AuthServer.Started.Load() {
			url := m.auth.GetAuthURL()
			m.copied = utils.CopyToClipboard(url) == nil
		}
		if key.Matches(msg, keyMap.PasteURL) && m.auth.AuthServer.Started.Load() {
			return m, m.focusPaste()
		}
	case tea.PasteMsg:
		if m.pasteFocused && !m.pasteSubmitting {
			var cmd tea.Cmd
			m.pasteInput, cmd = m.pasteInput.Update(msg)
			return m, cmd
		}
	case pasteResultMsg:
		m.pasteSubmitting = false
		if msg.err != nil {
			m.pasteErr = msg.err
			return m, nil
		}
		// The login flow completes through the token channel, same as the
		// browser callback.
		m.pasteErr = nil
		m.blurPaste()
		return m, nil
	case coreauth.AuthServerErr:
		m.err = msg.Err
		return m, m.quitAfterError()
	case authQuitMsg:
		return m, tea.Quit
	case State:
		m.authState = msg
		if msg == Authenticated {
			return m, nil
		}
		return m, m.listenForAuthUpdates
	}
	if m.pasteFocused && !m.pasteSubmitting {
		var cmd tea.Cmd
		m.pasteInput, cmd = m.pasteInput.Update(msg)
		return m, cmd
	}
	return m, nil
}
