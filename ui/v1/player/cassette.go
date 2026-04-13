package player

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

type cassette struct {
	spokeLeft    spoke
	spokeRight   spoke
	playerStatus cassetteStatus
}

type cassetteStatus struct {
	Online     bool
	Playing    bool
	ShowVolume bool
	CurrentMs  int
	DurationMs int
	Volume     int
	VolumeMax  int
}

func newCassette() cassette {
	return cassette{
		spokeLeft:  newSpoke(),
		spokeRight: newSpoke(),
	}
}

func (c *cassette) NextFrame() {
	c.spokeLeft.NextFrame()
	c.spokeRight.NextFrame()
}

func (c *cassette) View() string {
	return lipgloss.NewCompositor(c.layers()...).Render()
}

func (c *cassette) layers() []*lipgloss.Layer {
	leftReelRaw := c.spokeLeft.View()
	rightReelRaw := c.spokeRight.View()

	reelStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightMagenta)
	leftReel := reelStyle.Render(leftReelRaw)
	rightReel := reelStyle.Render(rightReelRaw)

	label := cassetteLabel()
	subtitle := cassetteSubtitle()
	windowTrim := cassetteWindowTrim()
	tapeWindow := cassetteTapeWindow()
	writeProtect := cassetteWriteProtect()
	statusIndicator := cassetteStatusIndicator(c.playerStatus)

	leftReelW, leftReelH := lipgloss.Width(leftReelRaw), lipgloss.Height(leftReelRaw)
	windowTrimW, windowTrimH := lipgloss.Width(windowTrim), lipgloss.Height(windowTrim)
	tapeWindowW, tapeWindowH := lipgloss.Width(tapeWindow), lipgloss.Height(tapeWindow)
	labelW := lipgloss.Width(label)
	subtitleW := lipgloss.Width(subtitle)
	writeProtectW := lipgloss.Width(writeProtect)
	statusIndicatorW := lipgloss.Width(statusIndicator)

	sidePad := 3
	centerGap := maxInt(windowTrimW+4, tapeWindowW+6, 12)
	innerW := sidePad*2 + leftReelW + centerGap + leftReelW
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
	statusIndicatorX := 1 + (innerW-statusIndicatorW)/2
	statusIndicatorY := shellH - 4

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
		lipgloss.NewLayer(statusIndicator).X(statusIndicatorX).Y(statusIndicatorY).ID("status-indicator"),
		lipgloss.NewLayer(writeProtect).X(writeProtectX).Y(writeProtectY).ID("write-protect"),
		lipgloss.NewLayer(cassetteScrew()).X(screwOffsetX).Y(topScrewY).ID("screw-tl"),
		lipgloss.NewLayer(cassetteScrew()).X(shellW - screwOffsetX - 1).Y(topScrewY).ID("screw-tr"),
		lipgloss.NewLayer(cassetteScrew()).X(screwOffsetX).Y(bottomScrewY).ID("screw-bl"),
		lipgloss.NewLayer(cassetteScrew()).X(shellW - screwOffsetX - 1).Y(bottomScrewY).ID("screw-br"),
	}
}

func cassetteStatusIndicator(status cassetteStatus) string {
	if !status.Online {
		dot := lipgloss.NewStyle().Foreground(lipgloss.BrightYellow).Bold(true).Render("●")
		text := lipgloss.NewStyle().Foreground(lipgloss.BrightYellow).Bold(true).Render(" GETTING READY")
		return dot + text
	}
	if status.ShowVolume {
		bar := volumeBar(status.Volume, status.VolumeMax, 10)
		text := fmt.Sprintf("[%s] %d%%", bar, volumePercent(status.Volume, status.VolumeMax))
		return lipgloss.NewStyle().Foreground(lipgloss.BrightGreen).Bold(true).Render(text)
	}
	if status.Playing {
		text := lipgloss.JoinHorizontal(lipgloss.Left, formatDuration(status.CurrentMs), "/", formatDuration(status.DurationMs))
		return lipgloss.NewStyle().Foreground(lipgloss.BrightGreen).Bold(true).Render(text)
	}
	dot := lipgloss.NewStyle().Foreground(lipgloss.BrightGreen).Bold(true).Render("●")
	text := lipgloss.NewStyle().Foreground(lipgloss.BrightGreen).Bold(true).Render(" READY")
	return dot + text
}

func formatDuration(ms int) string {
	if ms <= 0 {
		return "0:00"
	}
	totalSeconds := ms / 1000
	minutes := totalSeconds / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func volumePercent(volume, maxVolume int) int {
	if maxVolume <= 0 {
		maxVolume = 100
	}
	volume = max(0, min(volume, maxVolume))
	return int(float64(volume) * 100 / float64(maxVolume))
}

func volumeBar(volume, maxVolume, width int) string {
	if maxVolume <= 0 {
		maxVolume = 100
	}
	if width <= 0 {
		width = 10
	}
	volume = max(0, min(volume, maxVolume))
	filled := int(float64(volume) * float64(width) / float64(maxVolume))
	return strings.Repeat("■", filled) + strings.Repeat("□", width-filled)
}

func cassetteShell(innerW, innerH int) string {
	lines := make([]string, 0, innerH+2)
	lines = append(lines, "╭─"+strings.Repeat("═", innerW-2)+"─╮")
	for i := 0; i < innerH; i++ {
		fill := strings.Repeat(" ", innerW)
		if i == 0 {
			fill = "┄" + strings.Repeat("┈", innerW-2) + "┄"
		}
		if i == innerH-1 {
			fill = strings.Repeat("─", innerW)
		}
		lines = append(lines, "│"+fill+"│")
	}
	lines = append(lines, "╰─"+strings.Repeat("═", innerW-2)+"─╯")
	return lipgloss.NewStyle().Foreground(lipgloss.BrightCyan).Render(strings.Join(lines, "\n"))
}

func cassetteLabel() string {
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.BrightMagenta).
		Background(lipgloss.Black).
		Bold(true)
	accentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.BrightYellow).
		Background(lipgloss.Black).
		Bold(true)
	return accentStyle.Render(" ★ ") +
		labelStyle.Render(" LAZYSPOTIFY ") +
		accentStyle.Render("C-60") +
		accentStyle.Render(" ★ ")
}

func cassetteSubtitle() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.White).
		Faint(true).
		Render("TYPE I  │  STEREO  │  SIDE A")
}

func cassetteWindowTrim() string {
	lines := []string{
		"╔═══╗        ╔═══╗",
		"║ ◎ ║ ╌╌╌╌╌╌ ║ ◎ ║",
		"╚═══╝        ╚═══╝",
	}
	return lipgloss.NewStyle().Foreground(lipgloss.BrightCyan).Render(strings.Join(lines, "\n"))
}

func cassetteTapeWindow() string {
	lines := []string{
		"╭──────────╮",
		"│▓▓░░░░░░▓▓│",
		"│▓░      ░▓│",
		"╰──────────╯",
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Yellow).Render(strings.Join(lines, "\n"))
}

func cassetteWriteProtect() string {
	badgeStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightYellow).Bold(true)
	bracketStyle := lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	return bracketStyle.Render("⟦ ") + badgeStyle.Render("CHROME") + bracketStyle.Render(" · IEC II ⟧")
}

func cassetteScrew() string {
	return lipgloss.NewStyle().Foreground(lipgloss.BrightWhite).Render("x")
}

func maxInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	maxValue := values[0]
	for _, value := range values[1:] {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}
