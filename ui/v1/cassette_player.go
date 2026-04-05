package v1

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
)

type CassettePlayer struct {
	style lipgloss.Style
	cassette Cassette
	controls []PlayerButton
}
type ButtonKind int

const (
	PlayButton ButtonKind = iota
	PauseButton
	SeekForwardButton
	SeekBackwardButton
	NextButton
	PreviousButton
)

type PlayerButton struct {
	kind    ButtonKind
	style   lipgloss.Style
	icon    string
	pressed bool
}

func NewCassettePlayer() CassettePlayer {
	buttons := []PlayerButton{
		NewPlayButton(),
		NewPauseButton(),
		NewSeekForwardButton(),
		NewSeekBackwardButton(),
		NewNextButton(),
		NewPreviousButton(),
	}
	return CassettePlayer{
		style:     lipgloss.NewStyle(),
		cassette:  NewCassette(),
		controls:  buttons,
	}
}

func (c *CassettePlayer) View() string {
	cassette := c.cassette.View()
	cassetteW, cassetteH := lipgloss.Width(cassette), lipgloss.Height(cassette)
	var buttons string
	for _, button := range c.controls {
		buttons = lipgloss.JoinHorizontal(lipgloss.Left, buttons," ",button.View())
	}
	playerW := max(60, max(lipgloss.Width(buttons), cassetteW)+6)
	playerH := max(4, cassetteH + lipgloss.Height(buttons) +2)
	player := c.style.Render(playerShell(playerW, playerH))
	cassetteTapeX := playerW - cassetteW - 2
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(player).ID("player"),
		lipgloss.NewLayer(cassette).X(cassetteTapeX).Y(2).ID("cassette"),
		lipgloss.NewLayer(buttons).X(cassetteTapeX + 1).Y(2 + cassetteH).ID("buttons"),
	}
	compositor := lipgloss.NewCompositor(layers...)
	player = compositor.Render()
	return player
}

func playerShell(innerW int, innerH int) string {
	lines := make([]string, 0, innerH+2)
	// Top edge with rounded corners and decorative double-line accent
	lines = append(lines, "╭─"+strings.Repeat("─", innerW-2)+"─╮")
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
	lines = append(lines, "╰─"+strings.Repeat("─", innerW-2)+"─╯")

	return  lipgloss.JoinVertical(lipgloss.Left, lines...)
}

type borderChars struct {
	topL, topR, botL, botR, mid, fill string
}

func buttonShellBase(width int, height int, b borderChars) string {
	lines := make([]string, height)
	lines[0] = b.topL+strings.Repeat(b.fill, width-2)+b.topR
	for i := 1; i < height-1; i++ {
		lines[i] = b.mid + strings.Repeat(" ", width-2) + b.mid
	}
	lines[height-1] = b.botL+strings.Repeat(b.fill, width-2)+b.botR
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func buttonShell(width int, height int, playing bool) string {
	if(playing){
		return buttonShellBase(width, height, borderChars{"╔", "╗", "╚", "╝", "║", "═"})
	}
	return buttonShellBase(width, height, borderChars{"╭", "╮", "╰", "╯", "│", "─"})
}

func (c *CassettePlayer) NextFrame(playing bool){
	if(playing){
		c.cassette.NextFrame()
	}
}

func (c *CassettePlayer) NextButtonFrame(){
	for idx := range c.controls {
		if c.controls[idx].pressed {
			c.controls[idx].pressed = false
		}
	}
}

func (c *CassettePlayer) HandleButtonPress(kind ButtonKind) {
	c.controls[kind].pressed = true
}

func NewPlayButton() PlayerButton {
	return PlayerButton{
		kind:  PlayButton,
		style: lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370")),
		icon:  "|>",
	}
}

func NewPauseButton() PlayerButton {
	return PlayerButton{
		kind:  PauseButton,
		style: lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370")),
		icon:  "||",
	}
}

func NewSeekForwardButton() PlayerButton {
	return PlayerButton{
		kind:  SeekForwardButton,
		style: lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370")),
		icon:  ">>",
	}
}

func NewSeekBackwardButton() PlayerButton {
	return PlayerButton{
		kind:  SeekBackwardButton,
		style: lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370")),
		icon:  "<<",
	}
}

func NewNextButton() PlayerButton {
	return PlayerButton{
		kind:  NextButton,
		style: lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370")),
		icon:  ">|",
	}
}

func NewPreviousButton() PlayerButton {
	return PlayerButton{
		kind:  PreviousButton,
		style: lipgloss.NewStyle().Foreground(lipgloss.Color("#5C6370")),
		icon:  "|<",
	}
}

func (p *PlayerButton) View() string {
	const (
		btnW = 6
		btnH = 3
	)
	shell := buttonShell(btnW, btnH, p.pressed)
	if(p.pressed){
		logger.Log.Debug().Bool("pressed", p.pressed).Msg("pressed")
	}
	iconW := lipgloss.Width(p.icon)
	x := 1 + (btnW-2-iconW)/2
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(shell).ID("shell"),
		lipgloss.NewLayer(p.style.Render(p.icon)).X(x).Y(1).ID("icon"),
	}
	return lipgloss.NewCompositor(layers...).Render()
}

type NextButtonFrameMsg struct{}
func DoTickButtonPress() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return NextButtonFrameMsg{}
	})
}
