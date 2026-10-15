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
		head := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.BrightYellow).MarginBottom(1).Render("Authenticating with Spotify")
		msg := lipgloss.NewStyle().Width(m.width).MarginBottom(1).Render("Please open this link in your browser")
		styledURL := lipgloss.NewStyle().Width(m.width).Foreground(lipgloss.BrightBlue).Render(m.auth.GetAuthURL())
		hintText := "press c to copy"
		hintColor := lipgloss.BrightBlack
		if m.copied {
			hintText = "✓ copied to clipboard"
			hintColor = lipgloss.BrightGreen
		}
		hint := lipgloss.NewStyle().Foreground(hintColor).MarginTop(3).Render(hintText)
		remoteHint := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Render("browser can't reach this machine? after approving, copy the address bar url and press p to paste it")
		lines := []string{head, msg, styledURL, hint, remoteHint}
		if m.pasteFocused {
			input := lipgloss.NewStyle().MarginTop(2).Render(m.pasteInput.View())
			inputHint := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Render("enter: submit • esc: cancel")
			if m.pasteSubmitting {
				inputHint = "Completing login…"
			}
			lines = append(lines, input, inputHint)
		}
		if m.pasteErr != nil {
			lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.BrightRed).Render(m.pasteErr.Error()))
		}
		return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, lines...))
	}
	return tea.NewView("Authenticating with Spotify")
}
