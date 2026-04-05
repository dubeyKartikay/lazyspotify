package v1

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type Cassette struct {
	spokeLeft  Spoke
	spokeRight Spoke
}

func NewCassette() Cassette {
	return Cassette{
		spokeLeft:  NewSpoke(),
		spokeRight: NewSpoke(),
	}
}

func (c *Cassette) NextFrame() {
	c.spokeLeft.NextFrame()
	c.spokeRight.NextFrame()
}

func (c *Cassette) View() string {
	compositor := lipgloss.NewCompositor(c.cassetteLayers()...)
	return compositor.Render()
}

func (c *Cassette) cassetteLayers() []*lipgloss.Layer {
	leftReelRaw := c.spokeLeft.View()
	rightReelRaw := c.spokeRight.View()
	
	// Neon purple spinning reels
	reelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#C678DD"))
	leftReel := reelStyle.Render(leftReelRaw)
	rightReel := reelStyle.Render(rightReelRaw)
	
	label := cassetteLabel()
	subtitle := cassetteSubtitle()
	windowTrim := cassetteWindowTrim()
	tapeWindow := cassetteTapeWindow()
	writeProtect := cassetteWriteProtect()

	// Use unstyled widths for calculations to avoid ansi quirks, 
	// though lipgloss.Width handles ansi safely.
	leftReelW, leftReelH := lipgloss.Width(leftReelRaw), lipgloss.Height(leftReelRaw)
	windowTrimW, windowTrimH := lipgloss.Width(windowTrim), lipgloss.Height(windowTrim)
	tapeWindowW, tapeWindowH := lipgloss.Width(tapeWindow), lipgloss.Height(tapeWindow)
	labelW := lipgloss.Width(label)
	subtitleW := lipgloss.Width(subtitle)
	writeProtectW := lipgloss.Width(writeProtect)

	sidePad := 3
	centerGap := maxInt(windowTrimW+4, tapeWindowW+6, 12)
	innerW := sidePad*2 + leftReelW + centerGap + leftReelW + 1
	innerW = maxInt(innerW, labelW+6, subtitleW+6, writeProtectW+6)

	reelYInner := 3
	innerH := maxInt(reelYInner+leftReelH+3, 11)

	shell := cassetteShell(innerW, innerH)
	shellW := lipgloss.Width(shell)
	shellH := lipgloss.Height(shell)

	leftReelX := 1 + sidePad
	leftReelY := 1 + reelYInner
	rightReelX := shellW - 1 - sidePad - leftReelW
	rightReelY := leftReelY

	gapStart := leftReelX + leftReelW
	gapWidth := rightReelX - gapStart

	labelX := 1 + (innerW-labelW)/2
	labelY := 2
	subtitleX := 1 + (innerW-subtitleW)/2
	subtitleY := labelY + 1
	windowTrimX := gapStart + (gapWidth-windowTrimW)/2
	windowTrimY := leftReelY + (leftReelH-windowTrimH)/2 - 1
	tapeWindowX := gapStart + (gapWidth-tapeWindowW)/2
	tapeWindowY := leftReelY + (leftReelH-tapeWindowH)/2 + 1

	writeProtectX := 1 + (innerW-writeProtectW)/2
	writeProtectY := shellH - 3

	screwOffsetX := 2
	topScrewY := 1
	bottomScrewY := shellH - 3

	return []*lipgloss.Layer{
		lipgloss.NewLayer(shell).ID("shell"),
		lipgloss.NewLayer(label).X(labelX).Y(labelY).ID("label"),
		lipgloss.NewLayer(subtitle).X(subtitleX).Y(subtitleY).ID("subtitle"),
		lipgloss.NewLayer(leftReel).X(leftReelX).Y(leftReelY).ID("left-reel"),
		lipgloss.NewLayer(rightReel).X(rightReelX).Y(rightReelY).ID("right-reel"),
		lipgloss.NewLayer(windowTrim).X(windowTrimX).Y(windowTrimY).ID("window-trim"),
		lipgloss.NewLayer(tapeWindow).X(tapeWindowX).Y(tapeWindowY).ID("tape-window"),
		lipgloss.NewLayer(writeProtect).X(writeProtectX).Y(writeProtectY).ID("write-protect"),
		lipgloss.NewLayer(cassetteScrew()).X(screwOffsetX).Y(topScrewY).ID("screw-tl"),
		lipgloss.NewLayer(cassetteScrew()).X(shellW-screwOffsetX-1).Y(topScrewY).ID("screw-tr"),
		lipgloss.NewLayer(cassetteScrew()).X(screwOffsetX).Y(bottomScrewY).ID("screw-bl"),
		lipgloss.NewLayer(cassetteScrew()).X(shellW-screwOffsetX-1).Y(bottomScrewY).ID("screw-br"),
	}
}

func cassetteShell(innerW, innerH int) string {
	lines := make([]string, 0, innerH+2)
	// Top edge with rounded corners and decorative double-line accent
	lines = append(lines, "╭─"+strings.Repeat("═", innerW-2)+"─╮")
	for i := 0; i < innerH; i++ {
		fill := strings.Repeat(" ", innerW)
		if i == 0 {
			// Subtle inner ridge near the top
			fill = "┄" + strings.Repeat("┈", innerW-2) + "┄"
		}
		if i == innerH-1 {
			fill = strings.Repeat("─", innerW)
		}
		lines = append(lines, "│"+fill+"│")
	}
	lines = append(lines, "╰─"+strings.Repeat("═", innerW-2)+"─╯")
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370")).Render(strings.Join(lines, "\n"))
}

func cassetteLabel() string {
	// Retro cassette label with hot pink neon on dark chrome
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6AC1")). // hot pink neon
		Background(lipgloss.Color("#1E1E2E")). // deep dark bg
		Bold(true)
	accentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5C07B")). // warm gold accent
		Background(lipgloss.Color("#1E1E2E")).
		Bold(true)
	return accentStyle.Render(" ★ ") +
		labelStyle.Render(" LAZYSPOTIFY ") +
		accentStyle.Render("C-60") +
		accentStyle.Render(" ★ ")
}

func cassetteSubtitle() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ABB2BF")).
		Faint(true)
	return style.Render("TYPE I  │  STEREO  │  SIDE A")
}

func cassetteWindowTrim() string {
	lines := []string{
		"╔═══╗        ╔═══╗",
		"║ ◎ ║ ╌╌╌╌╌╌ ║ ◎ ║",
		"╚═══╝        ╚═══╝",
	}
	// Warm silver plastic bevel with slight teal tint
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#7EC8E3")).Render(strings.Join(lines, "\n"))
}

func cassetteTapeWindow() string {
	lines := []string{
		"╭────────╮",
		"│▓▓░░░░▓▓│",
		"│▓░    ░▓│",
		"╰────────╯",
	}
	// Deep amber magnetic tape window
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#D19A66")).Render(strings.Join(lines, "\n"))
}

func cassetteWriteProtect() string {
	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E5C07B")).
		Bold(true)
	bracketStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370"))
	return bracketStyle.Render("⟦ ") + badgeStyle.Render("CHROME") + bracketStyle.Render(" · IEC II ⟧")
}

func cassetteScrew() string {
	// Brushed chrome Phillips head screw
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#C8CCD4")).Render("x")
}

func maxInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}
