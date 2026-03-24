package v1

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dubeyKartikay/lazyspotify/core/auth"
)


type AuthModel struct {
	needsAuth bool
	auth      *auth.Authenticator
	authFlowUpdates chan string
	err       error
	width int
	height int
}

type statusMsg string


func newAuthModel() *AuthModel {
  return &AuthModel{
    needsAuth: false,
		auth: auth.New(),
    authFlowUpdates: make(chan string),
  }
}
func (m *AuthModel) startAuthFlow () tea.Msg{
	_,err := m.auth.ReAuthenticate(context.Background(),m.authFlowUpdates)
  if err != nil {
    return auth.AuthServerErr{Err: err}
  }
  return nil
}

func (m* AuthModel) listenForAuthUpdates() tea.Msg{
	updates := <- m.authFlowUpdates
	if updates == "success" {
		m.needsAuth = false
	}
  return statusMsg(updates)
}


func (m *AuthModel) Init() tea.Cmd {
	return nil
}

func (m *AuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.needsAuth && !m.auth.AuthServer.Started.Load() {
    return m, tea.Batch(m.startAuthFlow, m.listenForAuthUpdates)
	}

	switch msg := msg.(type) {
		case auth.AuthServerErr:
			m.err = msg.Err
			return m, tea.Quit
		case statusMsg:
			return m, m.listenForAuthUpdates

	}
  return m, nil
}

func (m *AuthModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("Error: cannot start auth server: %v", m.err))
	}
	if m.auth.AuthServer.Started.Load() {
		head := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.Color("yellow")).MarginBottom(1).Render("Authenticating with Spotify")
    msg := lipgloss.NewStyle().Width(m.width).MarginBottom(1).Render("Please open this link in your browser")
		styledUrl := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.Color("blue")).Render(m.auth.GetAuthURL())
	combinedView := lipgloss.JoinVertical(lipgloss.Left, head, msg, styledUrl)
		return tea.NewView(combinedView)
	}
	return tea.NewView("Authenticating with Spotify")
}

func (m *AuthModel) SetSize(width, height int) {
  m.width = width
  m.height = height
}


