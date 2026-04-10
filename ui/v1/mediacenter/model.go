package mediacenter

import (
	tea "charm.land/bubbletea/v2"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/displayscreen"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/mediapanel"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/player"
)

type Model struct {
	mediaPanel    mediapanel.Model
	player        player.Model
	displayScreen displayscreen.Model
	mediaListOpen bool
}

func NewModel() Model {
	return Model{
		mediaPanel:    mediapanel.NewModel(),
		player:        player.NewModel(),
		displayScreen: displayscreen.NewModel(),
	}
}

func (m *Model) SetDisplay(text string) {
	m.displayScreen.SetDisplay(text)
}

func (m *Model) SetDisplayFromSong(song common.SongInfo) {
	m.displayScreen.SetDisplayFromSong(song)
}

func (m *Model) UpdatePlayerStatus(status player.Status) {
	m.player.UpdateStatus(status)
}

func (m *Model) TickPlayer(playing bool) {
	m.player.NextFrame(playing)
}

func (m *Model) TickDisplay() tea.Cmd {
	return m.displayScreen.NextFrame()
}

func (m *Model) TickButtons() {
	m.player.NextButtonFrame()
}

func (m *Model) PressButton(kind player.ButtonKind) tea.Cmd {
	return m.player.HandleButtonPress(kind)
}

func (m *Model) ShowVolume() tea.Cmd {
	return m.player.ShowVolume()
}

func (m *Model) HideVolume() {
	m.player.HideVolume()
}

func (m *Model) StartLoading(kind common.ListKind) tea.Cmd {
	return m.mediaPanel.StartLoading(kind)
}

func (m *Model) SetContent(entities []common.Entity, kind common.ListKind, pagination common.PaginationInfo, request common.MediaRequest) tea.Cmd {
	return m.mediaPanel.SetContent(entities, kind, pagination, request)
}

func (m *Model) SetStatus(kind common.ListKind, message string) tea.Cmd {
	return m.mediaPanel.SetStatus(kind, message)
}

func (m *Model) CloseLibrary() {
	m.mediaListOpen = false
}

func (m *Model) IsOpen() bool {
	return m.mediaListOpen
}
