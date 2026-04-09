package v1

import (
	"os"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/ticker"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
)

func newModel() Model {
	return Model{
		mediaCenter: NewMediaCenter(),
		help:        newHelpModel(),
		keys:        newAppKeyMap(),
	}
}

func (m *Model) Init() tea.Cmd {
	cmd := func() tea.Msg {
		err := m.start()
		if err != nil && m.authModel.authState == Authenticated {
			return tea.Msg(err)
		}
		if m.authModel.authState == NeedsAuth {
			return tea.Msg(m.authModel.authState)
		}
		return startupCompleteMsg{}
	}

	return tea.Batch(cmd, ticker.StartTicker())
}

func (m *Model) View() tea.View {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	if m.authModel != nil && m.authModel.authState < Authenticated {
		return m.authModel.View()
	}
	mediaCenter := m.mediaCenter
	helpLine := helpStyle.Width(m.width).Align(lipgloss.Center).Render(m.help.View(m.keys))
	v := mediaCenter.View()
	modelView := lipgloss.NewStyle().Width(m.width).Height(m.height).Align(lipgloss.Center, lipgloss.Center).Render(v)
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(modelView).ID("model"),
		lipgloss.NewLayer(helpLine).Y(m.height - lipgloss.Height(helpLine)).ID("help"),
	}
	compositor := lipgloss.NewCompositor(layers...)
	modelView = compositor.Render()
	return tea.NewView(modelView)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	centerCmd := m.mediaCenter.Update(msg)

	switch msg := msg.(type) {
	case authState:
		if msg == Authenticated {
			logger.Log.Info().Msg("authenticated")
			return m, m.Init()
		}
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.ToggleHelp):
			m.help.ShowAll = !m.help.ShowAll
			return m, centerCmd
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		return m, centerCmd
	case ticker.TickFastMsg:
		cmd = m.NextFrame()
		return m, tea.Batch(cmd, centerCmd)
	case ticker.TickMsg:
		cmd = m.mediaCenter.displayScreen.NextFrame()
		return m, tea.Batch(cmd, centerCmd)
	case ticker.TickMsgVolume:
		m.mediaCenter.cassettePlayer.HideVolume()
		return m, centerCmd
	case ticker.TickClickMsg:
		cmd = m.NextButtonFrame()
		return m, tea.Batch(cmd, centerCmd)
	case MediaRequest:
		var startCmd tea.Cmd
		logger.Log.Info().Int("kind", int(msg.kind)).Str("cursor", msg.cursor).Int("page", msg.page).Msg("requesting media")
		if msg.showLoading {
			startCmd = m.mediaCenter.StartLoading()
		}
		fetchCmd := m.HandleMediaRequest(msg)
		return m, tea.Batch(startCmd, fetchCmd, centerCmd)
	case startupCompleteMsg:
		requestCmd := tea.Cmd(func() tea.Msg {
			return MediaRequestForListKind(Playlists)
		})
		return m, tea.Batch(m.waitForPlayerReady(), m.waitForPlayerEvent(), requestCmd, centerCmd)
	case playerReadyMsg:
		m.playerReady = true
		m.UpdateStatus()
		return m, centerCmd
	case playerReadyErrMsg:
		m.playerReady = false
		m.UpdateStatus()
		logger.Log.Error().Err(msg.err).Msg("failed to wait for player to be ready")
		return m, centerCmd
	case playerEventMsg:
		m.applyPlayerEvent(msg.event)
		m.UpdateStatus()
		return m, tea.Batch(m.waitForPlayerEvent(), centerCmd)
	case playerEventsClosedMsg:
		logger.Log.Warn().Msg("player events stream closed")
		return m, centerCmd
	case mediaLoadedMsg:
		setContentCmd := m.mediaCenter.SetContent(msg.entities, msg.kind, msg.pagination, msg.request)
		return m, tea.Batch(setContentCmd, centerCmd)
	case mediaLoadErrMsg:
		logger.Log.Error().Err(msg.err).Msg("failed to get user library")
		m.mediaCenter.StartLoading()
		return m, tea.Batch(m.mediaCenter.SetStatus("Failed to load library"), centerCmd)
	case playTrackErrMsg:
		logger.Log.Error().Err(msg.err).Msg("failed to play track")
		return m, tea.Batch(m.mediaCenter.SetStatus("Failed to play track"), centerCmd)
	case playTrackOkMsg:
		m.playing = true
		m.playerReady = true
		m.UpdateStatus()
		return m, tea.Batch(m.mediaCenter.SetStatus("Playing"), centerCmd)
	}

	if m.authModel != nil && m.authModel.authState < Authenticated  {
		newM, cmd := m.authModel.Update(msg)
		m.authModel = newM.(*AuthModel)
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.PlayPause):
			m.playing = !m.playing
			if m.playing {
				cmd = m.HandleButtonPress(PlayButton)

			} else {
				cmd = m.HandleButtonPress(PauseButton)
			}
			return m, tea.Batch(cmd, m.playPauseCmd(), centerCmd)
		case key.Matches(msg, m.keys.SeekForward):
			cmd = m.HandleButtonPress(SeekForwardButton)
			return m, tea.Batch(cmd, m.seekForwardCmd(), centerCmd)
		case key.Matches(msg, m.keys.SeekBackward):
			cmd = m.HandleButtonPress(SeekBackwardButton)
			return m, tea.Batch(cmd, m.seekBackwardCmd(), centerCmd)
		case key.Matches(msg, m.keys.NextTrack):
			cmd = m.HandleButtonPress(NextButton)
			return m, tea.Batch(cmd, m.nextCmd(), centerCmd)
		case key.Matches(msg, m.keys.PrevTrack):
			cmd = m.HandleButtonPress(PreviousButton)
			return m, tea.Batch(cmd, m.previousCmd(), centerCmd)
		case key.Matches(msg, m.keys.VolumeDown):
			m.markVolumeOverlay()
			m.previewVolume(-utils.GetConfig().Librespot.VolumeStep)
			return m, tea.Batch(cmd, m.decrementVolumeCmd(), centerCmd)
		case key.Matches(msg, m.keys.VolumeUp):
			m.markVolumeOverlay()
			m.previewVolume(utils.GetConfig().Librespot.VolumeStep)
			return m, tea.Batch(cmd, m.incrementVolumeCmd(), centerCmd)
		}
	}
	return m, centerCmd
}

func RunTui() {
	model := newModel()
	_, err := tea.NewProgram(&model).Run()
	model.shutdown()
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to run program")
		os.Exit(1)
	}
}
