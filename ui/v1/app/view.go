package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m *Model) View() tea.View {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	if m.authModel != nil && m.authModel.State() < 2 {
		return m.authModel.View()
	}

	mediaCenterView := m.mediaCenter.View()
	helpKeys := m.keys.WithMediaPanelOpen(m.mediaCenter.IsOpen())
	helpLine := helpStyle.Width(m.width).Align(lipgloss.Center).Render(m.help.View(helpKeys))
	modelView := lipgloss.NewStyle().Width(m.width).Height(m.height).Align(lipgloss.Center, lipgloss.Center).Render(mediaCenterView)
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(modelView).ID("model"),
		lipgloss.NewLayer(helpLine).Y(m.height - lipgloss.Height(helpLine)).ID("help"),
	}
	return tea.NewView(lipgloss.NewCompositor(layers...).Render())
}
