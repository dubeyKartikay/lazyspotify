package ticker

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

type TickFastMsg struct {}
type TickClickMsg struct {}
type TickMsg struct {}
type TickMsgVolume struct {}

func DoTickFast() tea.Cmd {
	return tea.Tick(180*time.Millisecond, func(t time.Time) tea.Msg {
		return TickFastMsg{}
	})
}

func DoTickClick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickClickMsg{}
	})
}

func DoTick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

func DoTickVolume() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return TickMsgVolume{}
	})
}

func StartTicker() tea.Cmd {
	return tea.Batch(DoTickFast(), DoTickClick(),DoTick())
}
