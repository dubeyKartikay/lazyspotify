package displayscreen

import (
	"charm.land/lipgloss/v2"
)

type Model struct {
	width        int
	height       int
	display      string
	scrollOffset int
	styles       styles
}

type styles struct {
	panel   lipgloss.Style
	primary lipgloss.Style
	accent  lipgloss.Style
	muted   lipgloss.Style
	marquee lipgloss.Style
}

func NewModel() Model {
	return Model{
		display: "Lazyspotify: The cutest terminal music player",
		styles: styles{
			panel: lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Cyan).
				Foreground(lipgloss.White),
			primary: lipgloss.NewStyle().Foreground(lipgloss.BrightYellow).Bold(true),
			accent:  lipgloss.NewStyle().Foreground(lipgloss.BrightCyan),
			muted:   lipgloss.NewStyle().Foreground(lipgloss.White),
			marquee: lipgloss.NewStyle().Foreground(lipgloss.BrightCyan).Bold(true),
		},
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) SetDisplay(s string) {
	m.display = s
	m.scrollOffset = 0
}
