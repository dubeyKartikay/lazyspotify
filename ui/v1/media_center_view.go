package v1

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func (m *MediaCenter) View(playerReady bool) string {
	cassette := m.cassettePlayer.View(playerReady)
	visibleList := m.lists.Peek()
	cassetteW, cassetteH := lipgloss.Size(cassette)
	listW := 30
	listH := cassetteH
	visibleList.SetSize(listW, listH)
	mediaList := visibleList.View(m.renderPanelNav())
	m.displayScreen.SetSize(listW+cassetteW, 3)
	v := lipgloss.JoinHorizontal(lipgloss.Top, mediaList, cassette)
	v = lipgloss.JoinVertical(lipgloss.Left, m.displayScreen.View(), v)
	shell := lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).Render(v)
	return shell
}

func (m *MediaCenter) renderPanelNav() string {
	visibleList := m.lists.Peek()
	rootList := m.lists.Items[0]
	segments := []struct {
		label string
		kind  ListKind
	}{
		{label: "PL", kind: Playlists},
		{label: "TR", kind: Tracks},
		{label: "AL", kind: Albums},
		{label: "AR", kind: Artists},
	}

	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		if rootList.kind == segment.kind {
			parts = append(parts, visibleList.styles.panelNavActive.Render(segment.label))
			continue
		}
		parts = append(parts, visibleList.styles.panelNavMuted.Render(segment.label))
	}

	return visibleList.styles.panelNav.Render(strings.Join(parts, " - "))
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
