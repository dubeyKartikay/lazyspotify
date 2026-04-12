package auth

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m *Model) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("Authentication failed: %v\nExiting...", m.err))
	}
	if m.auth.AuthServer.Started.Load() {
		head := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.Color("11")).MarginBottom(1).Render("Authenticating with Spotify")
		msg := lipgloss.NewStyle().Width(m.width).MarginBottom(1).Render("Please open this link in your browser")
		styledURL := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.Color("12")).Render(m.auth.GetAuthURL())
		hintText := "press c to copy"
		hintColor := lipgloss.Color("8")
		if m.copied {
			hintText = "✓ copied to clipboard"
			hintColor = lipgloss.Color("10")
		}
		hint := lipgloss.NewStyle().Foreground(hintColor).MarginTop(3).Render(hintText)
		return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, head, msg, styledURL, hint))
	}
	return tea.NewView("Authenticating with Spotify")
}
