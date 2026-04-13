package medialist

import (
	"charm.land/lipgloss/v2"
)

func (m Model) View() string {
	listWidth := m.width - 4
	listHeight := m.height - 1
	footer := ""
	if m.pager.TotalPages > 1 {
		footer = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Render(m.pager.View())
	}
	if listHeight < 1 {
		listHeight = 1
	}
	m.list.SetSize(listWidth, listHeight)
	if m.state == Loading {
		m.list.Title = "Loading..."
		m.list.SetItems(nil)
	}
	if footer == "" {
		return m.list.View()
	}
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(m.list.View()).ID("list"),
		lipgloss.NewLayer(footer).X(listWidth/2 - lipgloss.Width(footer)/2).Y(listHeight - lipgloss.Height(footer)).ID("footer"),
	}
	composed := lipgloss.NewCompositor(layers...)
	return composed.Render()
}
