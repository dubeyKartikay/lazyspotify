package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/lyricsview"
)

func (m *Model) View() tea.View {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	if m.fatalErr != nil {
		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.BrightRed).Render("Error")
		message := lipgloss.NewStyle().MarginTop(1).Align(lipgloss.Center).Render(fmt.Sprintf("%v", m.fatalErr))
		hint := lipgloss.NewStyle().MarginTop(1).Foreground(lipgloss.BrightBlack).Render("Exiting...")
		content := lipgloss.JoinVertical(lipgloss.Center, title, message, hint)
		view := lipgloss.NewStyle().Width(m.width).Height(m.height).Align(lipgloss.Center, lipgloss.Center).Render(content)
		return tea.NewView(view)
	}
	if m.authModel != nil && m.authModel.State() < 2 {
		return m.authModel.View()
	}

	var mediaCenterView string
	if m.lyricsViewOpen {
		mediaCenterView = m.renderLyricsOverlay()
	} else {
		mediaCenterView = m.mediaCenter.View(m.width, m.height)
	}
	helpKeys := m.keys.WithMediaPanelOpen(m.mediaCenter.IsOpen()).WithInfoOpen(m.mediaCenter.InfoOpen())
	helpLine := helpStyle.Width(m.width).Align(lipgloss.Center).Render(m.help.View(helpKeys))
	if m.viewportTooSmall(mediaCenterView, helpLine) {
		return tea.NewView(m.smallViewportView(mediaCenterView, helpLine))
	}
	modelView := lipgloss.NewStyle().Width(m.width).Height(m.height).Align(lipgloss.Center, lipgloss.Center).Render(mediaCenterView)
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(modelView).ID("model"),
	}
	if(!m.mediaCenter.IsZenMode()){
		layers = append(layers,lipgloss.NewLayer(helpLine).Y(m.height - lipgloss.Height(helpLine)).ID("help"))
	}
	return tea.NewView(lipgloss.NewCompositor(layers...).Render())
}

func (m *Model) renderLyricsOverlay() string {
	display := m.mediaCenter.DisplayView(m.width)
	player := m.mediaCenter.PlayerView()
	dispH := lipgloss.Height(display)
	playerH := lipgloss.Height(player)
	bodyH := m.height - dispH - playerH - 1 // -1 for help line
	if bodyH < 3 {
		bodyH = 3
	}
	state := lyricsview.State{
		Lines:    m.lyricsLines,
		Idx:      m.currentLyricIdx(),
		SyncType: m.lyricsSync,
		Err:      m.lyricsErr,
		Song:     m.songInfo,
	}
	body := lyricsview.Render(state, m.width, bodyH)
	return lipgloss.JoinVertical(lipgloss.Left, display, body, player)
}

func (m *Model) viewportTooSmall(mediaCenterView, helpLine string) bool {
	if m.width <= 0 || m.height <= 0 {
		return false
	}
	requiredWidth := lipgloss.Width(mediaCenterView)
	requiredHeight := lipgloss.Height(mediaCenterView) + lipgloss.Height(helpLine)
	return m.width < requiredWidth || m.height < requiredHeight
}

func (m *Model) smallViewportView(mediaCenterView, helpLine string) string {
	requiredWidth := lipgloss.Width(mediaCenterView)
	requiredHeight := lipgloss.Height(mediaCenterView) + lipgloss.Height(helpLine)
	message := lipgloss.JoinVertical(
		lipgloss.Center,
		"terminal size too small",
		lipgloss.NewStyle().Foreground(lipgloss.BrightBlack).Render(
			fmt.Sprintf("need at least %dx%d", requiredWidth, requiredHeight),
		),
	)
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(message)
}
