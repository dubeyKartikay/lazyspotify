package common

import (
	"charm.land/bubbles/v2/key"
)

type AppKeyMap struct {
	ToggleHelp     key.Binding
	Quit           key.Binding
	CycleLibrary   key.Binding
	Search         key.Binding
	Cancel         key.Binding
	MoreInfo       key.Binding
	InfoScrollUp   key.Binding
	InfoScrollDown key.Binding
	Select         key.Binding
	Back           key.Binding
	NextPage       key.Binding
	PrevPage       key.Binding
	TogglePanel    key.Binding
	PlayPause      key.Binding
	SeekForward    key.Binding
	SeekBackward   key.Binding
	NextTrack      key.Binding
	PrevTrack      key.Binding
	VolumeDown     key.Binding
	VolumeUp       key.Binding
	MediaPanelOpen bool
	InfoOpen       bool
}

func NewAppKeyMap() AppKeyMap {
	return AppKeyMap{
		ToggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		CycleLibrary: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next panel"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		MoreInfo: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "more info"),
		),
		InfoScrollUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "scroll info up"),
		),
		InfoScrollDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "scroll info down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select item"),
		),
		Back: key.NewBinding(
			key.WithKeys("backspace", "delete"),
			key.WithHelp("del", "back"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("right", "l", "]"),
			key.WithHelp("right/l/]", "next page"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("left", "h", "["),
			key.WithHelp("left/h/[", "prev page"),
		),
		TogglePanel: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle panel"),
		),
		PlayPause: key.NewBinding(
			key.WithKeys(" ", "space", "p"),
			key.WithHelp("space/p", "play pause"),
		),
		SeekForward: key.NewBinding(
			key.WithKeys("right", "l", "ctrl+f", "]"),
			key.WithHelp("right/l/]", "seek +"),
		),
		SeekBackward: key.NewBinding(
			key.WithKeys("left", "h", "ctrl+b", "["),
			key.WithHelp("left/h/[", "seek -"),
		),
		NextTrack: key.NewBinding(
			key.WithKeys("n", "ctrl+s"),
			key.WithHelp("n", "next"),
		),
		PrevTrack: key.NewBinding(
			key.WithKeys("N", "ctrl+r"),
			key.WithHelp("N", "previous"),
		),
		VolumeDown: key.NewBinding(
			key.WithKeys("j", "ctrl+p"),
			key.WithHelp("j", "volume -"),
		),
		VolumeUp: key.NewBinding(
			key.WithKeys("k", "ctrl+n"),
			key.WithHelp("k", "volume +"),
		),
	}
}

func (k AppKeyMap) ShortHelp() []key.Binding {
	bindings := []key.Binding{k.ToggleHelp, k.Quit, k.TogglePanel}
	return bindings
}

func (k AppKeyMap) FullHelp() [][]key.Binding {
	help := [][]key.Binding{
		{k.ToggleHelp, k.Quit, k.TogglePanel},
	}
	if k.MediaPanelOpen {
		if k.InfoOpen {
			closeInfo := key.NewBinding(
				key.WithKeys("i", "esc", "backspace", "delete"),
				key.WithHelp("i/esc/del", "close info"),
			)
			scrollInfo := key.NewBinding(
				key.WithKeys("ctrl+u", "ctrl+d"),
				key.WithHelp("ctrl+u/d", "scroll info"),
			)
			return append(help,
				[]key.Binding{k.CycleLibrary, k.Search, k.Select, closeInfo},
				[]key.Binding{k.NextPage, k.PrevPage, scrollInfo},
			)
		}
		return append(help, []key.Binding{k.CycleLibrary, k.Search, k.MoreInfo, k.Select, k.Back, k.NextPage, k.PrevPage})
	}
	return append(help,
		[]key.Binding{k.PlayPause, k.SeekForward, k.SeekBackward},
		[]key.Binding{k.VolumeDown, k.VolumeUp, k.NextTrack, k.PrevTrack},
	)
}

func (k AppKeyMap) WithMediaPanelOpen(open bool) AppKeyMap {
	k.MediaPanelOpen = open
	return k
}

func (k AppKeyMap) WithInfoOpen(open bool) AppKeyMap {
	k.InfoOpen = open
	return k
}
