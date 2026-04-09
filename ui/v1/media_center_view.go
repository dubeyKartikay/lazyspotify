package v1

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m *MediaCenter) View() string {
	cassette := m.cassettePlayer.View()
	cassetteW, cassetteH := lipgloss.Size(cassette)
	var mediaList string
	var listH int
	var listW int
	if m.mediaListOpen {
		listW = 30
		listH = cassetteH
		m.mediaPanel.SetSize(listW, listH)
		mediaList = m.mediaPanel.View()
	}
	m.displayScreen.SetSize(listW+cassetteW, 3)
	v := lipgloss.JoinHorizontal(lipgloss.Top, mediaList, cassette)
	v = lipgloss.JoinVertical(lipgloss.Left, m.displayScreen.View(), v)
	shell := lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).Render(v)
	return shell
}

func outerShell(width int, height int) string {
	lines := make([]string, 0, height+2)
	lines = append(lines, strings.Repeat(" ", width))
	for range height {
		fill := strings.Repeat(" ", width)
		lines = append(lines, fill)
	}
	lines = append(lines, strings.Repeat(" ", width))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
