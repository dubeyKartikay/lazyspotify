package mediacenter

import (
	"charm.land/lipgloss/v2"
)

func (m *Model) View(maxW, maxH int) string {
	playerView := m.player.View(m.zenMode)
	playerW, playerH := lipgloss.Size(playerView)
	var mediaList string
	listW := 0
	if m.mediaListOpen {
		listW = 30
		m.mediaPanel.SetSize(listW, playerH)
		mediaList = m.mediaPanel.View()
	}
	m.displayScreen.SetSize(listW+playerW, 3)
	content := lipgloss.JoinHorizontal(lipgloss.Top, mediaList, playerView)
	if !m.zenMode {
		content = lipgloss.JoinVertical(lipgloss.Left, m.displayScreen.View(), content)
	}
	v := lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).Render(content)
	w, h := lipgloss.Size(v)
	if ((w > maxW) || (h > maxH)) && m.mediaListOpen {
		return lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).Render(mediaList)
	}
	return v
}
