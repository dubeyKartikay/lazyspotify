package v1

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dubeyKartikay/lazyspotify/core/auth"
)

var authKeyMap = struct {
	CopyURL key.Binding
}{
	CopyURL: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy url")),
}
type authState int
const (
	NeedsAuth authState = iota
	Authenticating
	Authenticated
)
type AuthModel struct {
	authState       authState
	auth            *auth.Authenticator
	authFlowUpdates chan string
	err             error
	width           int
	height          int
	copied          bool
}


func newAuthModel() *AuthModel {
	return &AuthModel{
		authState:       Authenticated,
		auth:            auth.New(),
		authFlowUpdates: make(chan string),
	}
}
func (m *AuthModel) startAuthFlow() tea.Msg {
	_, err := m.auth.ReAuthenticate(context.Background(), m.authFlowUpdates)
	if err != nil {
		return auth.AuthServerErr{Err: err}
	}
	return nil
}

func (m *AuthModel) listenForAuthUpdates() tea.Msg {
	updates := <-m.authFlowUpdates
	if updates == "success" {
		m.authState = Authenticated
	}
	return m.authState
}

func (m *AuthModel) Init() tea.Cmd {
	return nil
}

func (m *AuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.authState == NeedsAuth {
		m.authState = Authenticating
		return m, tea.Batch(m.startAuthFlow, m.listenForAuthUpdates)
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, authKeyMap.CopyURL) && m.auth.AuthServer.Started.Load() {
			url := m.auth.GetAuthURL()
			cmd := exec.Command("pbcopy")
			cmd.Stdin = strings.NewReader(url)
			m.copied = cmd.Run() == nil
		}
	case auth.AuthServerErr:
		m.err = msg.Err
		return m, tea.Quit
	case authState:
		return m, m.listenForAuthUpdates
	}
	return m, nil
}

func (m *AuthModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("Error: cannot start auth server: %v", m.err))
	}
	if m.auth.AuthServer.Started.Load() {
		head := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.Color("11")).MarginBottom(1).Render("Authenticating with Spotify")
		msg := lipgloss.NewStyle().Width(m.width).MarginBottom(1).Render("Please open this link in your browser")
		styledUrl := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.Color("12")).Render(m.auth.GetAuthURL())
		var hintText string
		if m.copied {
			hintText = "✓ copied to clipboard"
		} else {
			hintText = "press c to copy"
		}
		hintColor := lipgloss.Color("8")
		if m.copied {
			hintColor = lipgloss.Color("10")
		}
		hint := lipgloss.NewStyle().Foreground(hintColor).MarginTop(3).Render(hintText)
		combinedView := lipgloss.JoinVertical(lipgloss.Left, head, msg, styledUrl, hint)
		return tea.NewView(combinedView)
	}
	return tea.NewView("Authenticating with Spotify")
}

func (m *AuthModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}
