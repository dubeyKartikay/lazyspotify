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

var keyMap = struct {
	CopyURL key.Binding
}{
	CopyURL: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy url")),
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
		m.authState = Authenticated
	}
	return m.authState
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.authState == NeedsAuth {
		m.authState = Authenticating
		return m, tea.Batch(m.startAuthFlow, m.listenForAuthUpdates)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, keyMap.CopyURL) && m.auth.AuthServer.Started.Load() {
			url := m.auth.GetAuthURL()
			m.copied = utils.CopyToClipboard(url) == nil
		}
	case coreauth.AuthServerErr:
		m.err = msg.Err
		return m, m.quitAfterError()
	case authQuitMsg:
		return m, tea.Quit
	case State:
		return m, m.listenForAuthUpdates
	}
	return m, nil
}
