package lyricsview

import (
	"strings"

	"charm.land/lipgloss/v2"
	spotifylyrics "github.com/dubeyKartikay/lazyspotify/spotify/lyrics"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
)

// State is everything the lyrics screen needs to render. Held by the app and
// passed into Render — the package itself is stateless so we can recompute
// freely on resize / tick without ownership concerns.
type State struct {
	Lines    []spotifylyrics.Line
	Idx      int
	SyncType string
	Err      string
	Song     common.SongInfo
}

const (
	headerRows = 2
	footerRows = 2
	sidePad    = 2
)

// Render produces a single string sized to (width, height). Intended to fill
// the body area between the now-playing strip and the player controls; the
// caller is responsible for assembling those.
func Render(s State, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	bodyW := width - 2*sidePad
	if bodyW < 8 {
		bodyW = max(width, 1)
	}

	header := renderHeader(s, bodyW)
	footer := renderFooter(s, bodyW)

	bodyH := height - headerRows - footerRows
	if bodyH < 1 {
		bodyH = 1
	}

	body := renderBody(s, bodyW, bodyH)

	pad := strings.Repeat(" ", sidePad)
	indent := func(block string) string {
		lines := strings.Split(block, "\n")
		for i, ln := range lines {
			lines[i] = pad + ln
		}
		return strings.Join(lines, "\n")
	}

	parts := []string{
		indent(header),
		indent(body),
		indent(footer),
	}
	return strings.Join(parts, "\n")
}

func renderHeader(s State, width int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231"))
	subStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	title := strings.TrimSpace(s.Song.Title)
	artist := strings.TrimSpace(s.Song.Artist)
	if title == "" {
		title = "—"
	}
	if artist == "" {
		artist = ""
	}

	titleLine := truncateToWidth(title, width)
	subLine := truncateToWidth(artist, width)
	return titleStyle.Render(padRight(titleLine, width)) + "\n" + subStyle.Render(padRight(subLine, width))
}

func renderFooter(s State, width int) string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hint := "press L to close"
	left := s.SyncType
	if left == "" && len(s.Lines) > 0 {
		left = "LINE_SYNCED"
	}
	gap := width - len(left) - len(hint)
	if gap < 1 {
		gap = 1
	}
	line := truncateToWidth(left+strings.Repeat(" ", gap)+hint, width)
	return "\n" + dim.Render(padRight(line, width))
}

// renderBody lays out the visible lyric rows. We expand each source line into
// wrapped visual rows, then pick a window of `height` rows centered roughly on
// the rows belonging to the current line.
func renderBody(s State, width, height int) string {
	if s.Err != "" {
		return centerMessage("Lyrics: "+s.Err, width, height)
	}
	if len(s.Lines) == 0 {
		return centerMessage("Lyrics: loading or unavailable for this track.", width, height)
	}

	type visualRow struct {
		text     string
		srcIdx   int
		isAnchor bool // first wrapped row of its source line
	}
	rows := make([]visualRow, 0, len(s.Lines))
	for i, ln := range s.Lines {
		wrapped := wrapLine(ln.Words, width)
		for j, w := range wrapped {
			rows = append(rows, visualRow{text: w, srcIdx: i, isAnchor: j == 0})
		}
	}

	// find first row of current source line
	cursorRow := 0
	for i, r := range rows {
		if r.srcIdx == s.Idx && r.isAnchor {
			cursorRow = i
			break
		}
	}

	start := cursorRow - height/2
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(rows) {
		end = len(rows)
		start = max(0, end-height)
	}

	current := lipgloss.NewStyle().Foreground(lipgloss.Color("228")).Bold(true)
	near := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	far := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	out := make([]string, 0, height)
	for i := start; i < end; i++ {
		r := rows[i]
		text := padRight(r.text, width)
		var styled string
		switch {
		case r.srcIdx == s.Idx:
			styled = current.Render(text)
		case absInt(r.srcIdx-s.Idx) <= 1:
			styled = near.Render(text)
		default:
			styled = far.Render(text)
		}
		out = append(out, styled)
	}
	for len(out) < height {
		out = append(out, padRight("", width))
	}
	return strings.Join(out, "\n")
}

func centerMessage(msg string, width, height int) string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	body := truncateToWidth(msg, width)
	mid := height / 2
	if mid < 0 {
		mid = 0
	}
	out := make([]string, height)
	for i := 0; i < height; i++ {
		if i == mid {
			out[i] = dim.Render(padRight(body, width))
		} else {
			out[i] = padRight("", width)
		}
	}
	return strings.Join(out, "\n")
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
