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
	topW := listW + playerW
	m.displayScreen.SetSize(topW, 3)
	row := lipgloss.JoinHorizontal(lipgloss.Top, mediaList, playerView)

	var v string
	if m.zenMode {
		v = row
	} else {
		v = lipgloss.JoinVertical(lipgloss.Left, m.displayScreen.View(), row)
	}
	w, h := lipgloss.Size(v)

	if (w > maxW || h > maxH) && m.mediaListOpen {
		compact := lipgloss.JoinHorizontal(lipgloss.Top, mediaList, playerView)
		cw, ch := lipgloss.Size(compact)
		if cw <= maxW && ch <= maxH {
			return lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).Render(compact)
		}
		pw, ph := lipgloss.Size(playerView)
		if pw <= maxW && ph <= maxH {
			return lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).Render(playerView)
		}
		v = mediaList
	}
	return lipgloss.NewStyle().BorderStyle(lipgloss.HiddenBorder()).Render(v)
}
