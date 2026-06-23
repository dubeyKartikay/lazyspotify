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
	zenMode       bool
	keys          common.AppKeyMap
}

func NewModel(keys common.AppKeyMap) Model {
	return Model{
		mediaPanel:    mediapanel.NewModel(keys),
		player:        player.NewModel(),
		displayScreen: displayscreen.NewModel(),
		keys:          keys,
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
	m.mediaPanel.CloseInfo()
}

func (m *Model) IsOpen() bool {
	return m.mediaListOpen
}

func (m *Model) InfoOpen() bool {
	return m.mediaPanel.InfoOpen()
}

func (m *Model) IsZenMode() bool {
	return m.zenMode
}

// DisplayView renders the now-playing strip sized to width. Used by the
// lyrics overlay to keep the same header chrome visible.
func (m *Model) DisplayView(width int) string {
	m.displayScreen.SetSize(width, 3)
	return m.displayScreen.View()
}

// PlayerView renders the player controls strip honoring zen mode. Used by
// the lyrics overlay so the same controls remain visible underneath.
func (m *Model) PlayerView() string {
	return m.player.View(m.zenMode)
}
