package v1

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/ticker"
)

type displayScreen struct {
	songInfo       SongInfo
	width          int
	height         int
	defaultDisplay string
	scrollOffset   int
	styles         displayStyles
}

type displayStyles struct {
	panel   lipgloss.Style
	primary lipgloss.Style
	accent  lipgloss.Style
	muted   lipgloss.Style
	marquee lipgloss.Style
}

func newDisplayScreen() displayScreen {
	return displayScreen{
		defaultDisplay: "Lazyspotify: The cutest terminal music player, Lazyspotify: The cutest terminal music player, Lazyspotify: The cutest terminal music player",
		styles: displayStyles{
			panel: lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("30")).
				Foreground(lipgloss.Color("229")),
			primary: lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true),
			accent:  lipgloss.NewStyle().Foreground(lipgloss.Color("50")),
			muted:   lipgloss.NewStyle().Foreground(lipgloss.Color("151")),
			marquee: lipgloss.NewStyle().Foreground(lipgloss.Color("195")).Bold(true),
		},
	}
}

func (d *displayScreen) SetSongInfo(songInfo SongInfo) {
	d.songInfo = songInfo
}

func (d *displayScreen) View() string {
	songInfo := d.songInfo
	raw := d.defaultDisplay
	styled := d.styles.muted.Render(raw)
	if songInfo.title != "" {
		separator := " • "
		raw = songInfo.title + separator + songInfo.artist + separator + songInfo.album
		styled = lipgloss.JoinHorizontal(
			lipgloss.Left,
			d.styles.primary.Render(songInfo.title),
			d.styles.accent.Render(separator),
			d.styles.muted.Render(songInfo.artist),
			d.styles.accent.Render(separator),
			d.styles.muted.Render(songInfo.album),
		)
	}

	contentWidth := max(0, d.width-2)
	if contentWidth > 0 {
		if lipgloss.Width(raw) > contentWidth {
			styled = d.styles.marquee.Render(d.scrollText(raw, contentWidth))
		}
	}
	panel := d.styles.panel.Width(d.width).Height(d.height).Render(styled)
	return panel
}

func (d *displayScreen) SetSize(width int, height int) {
	d.width = width
	d.height = height
}

func (d *displayScreen) NextFrame() tea.Cmd {
	d.scrollOffset++
	logger.Log.Debug().Int("scrollOffset", d.scrollOffset).Msg("scrolling")
	return ticker.DoTick()
}

func (d *displayScreen) scrollText(text string, width int) string {
	if width <= 0 {
		return ""
	}

	if lipgloss.Width(text) <= width {
		return text
	}

	const gap = "   "
	base := []rune(text + gap)
	track := append(base, base...)
	if len(base) == 0 {
		return strings.Repeat(" ", width)
	}

	start := d.scrollOffset % len(base)
	end := start + width
	if end > len(track) {
		end = len(track)
	}

	visible := string(track[start:end])
	if len([]rune(visible)) < width {
		visible += strings.Repeat(" ", width-len([]rune(visible)))
	}

	return visible
}
