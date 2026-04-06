package v1

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/ticker"
)

func newModel() Model {
	return Model{
		mediaCenter: NewMediaCenter(),
	}
}

func (m *Model) Init() tea.Cmd {
	cmd := func() tea.Msg {
		err := m.start()
		if err != nil && !m.authModel.needsAuth {
			return tea.Msg(err)
		}
		if m.authModel.needsAuth {
			return tea.Msg(m.authModel.needsAuth)
		}
		return startupCompleteMsg{}
	}

	return tea.Batch(cmd, ticker.StartTicker())
}

func (m *Model) View() tea.View {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	if m.authModel != nil && m.authModel.needsAuth {
		return m.authModel.View()
	}
	mediaCenter := m.mediaCenter
	v := mediaCenter.View(m.playerReady)
	return tea.NewView(v + "\n" + helpStyle.Render("Press q to quit"))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	centerCmd := m.mediaCenter.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
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
	case ticker.TickSlowMsg:
		cmd = m.NextButtonFrame()
		return m, tea.Batch(cmd, centerCmd)
	case MediaRequest:
		var startCmd tea.Cmd
		if msg.showLoading {
			startCmd = m.mediaCenter.StartLoading()
		}
		fetchCmd := m.HandleMediaRequest(msg)
		return m, tea.Batch(startCmd, fetchCmd, centerCmd)
	case startupCompleteMsg:
		requestCmd := tea.Cmd(func() tea.Msg {
			return MediaRequestForListKind(Playlists, 0)
		})
		return m, tea.Batch(m.waitForPlayerReady(), m.waitForPlayerEvent(), requestCmd, centerCmd)
	case playerReadyMsg:
		m.playerReady = true
		m.playDailyMix()
		return m, centerCmd
	case playerReadyErrMsg:
		m.playerReady = false
		logger.Log.Error().Err(msg.err).Msg("failed to wait for player to be ready")
		return m, centerCmd
	case playerEventMsg:
		m.applyPlayerEvent(msg.event)
		return m, tea.Batch(m.waitForPlayerEvent(), centerCmd)
	case playerEventsClosedMsg:
		logger.Log.Warn().Msg("player events stream closed")
		return m, centerCmd
	case mediaLoadedMsg:
		setContentCmd := m.mediaCenter.SetContent(msg.entities, msg.kind)
		return m, tea.Batch(setContentCmd, centerCmd)
	case mediaLoadErrMsg:
		logger.Log.Error().Err(msg.err).Msg("failed to get user library")
		m.mediaCenter.StopSpinner()
		return m, tea.Batch(m.mediaCenter.SetStatus("Failed to load library"), centerCmd)
	case playTrackErrMsg:
		logger.Log.Error().Err(msg.err).Msg("failed to play track")
		return m, tea.Batch(m.mediaCenter.SetStatus("Failed to play track"), centerCmd)
	case playTrackOkMsg:
		m.playing = true
		return m, tea.Batch(m.mediaCenter.SetStatus("Playing"), centerCmd)
	}
	if m.authModel != nil && m.authModel.needsAuth {
		newM, cmd := m.authModel.Update(msg)
		m.authModel = newM.(*AuthModel)
		return m, tea.Batch(cmd, centerCmd)
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "tab":
			nextKind := m.mediaCenter.NextListKind()
			requestCmd := tea.Cmd(func() tea.Msg {
				m.mediaCenter.lists.Items = nil
				m.mediaCenter.lists.Push(newMediaList())
				return MediaRequestForListKind(nextKind, 0)
			})
			return m, tea.Batch(requestCmd, centerCmd)
		case " ", "p":
			m.playing = !m.playing
			if m.playing {
				cmd = m.HandleButtonPress(PlayButton)

			} else {
				cmd = m.HandleButtonPress(PauseButton)
			}
			m.playPause()
			return m, tea.Batch(cmd, centerCmd)
		case "right", "l", "ctrl+f", "]":
			cmd = m.HandleButtonPress(SeekForwardButton)
			m.seekForward()
			return m, tea.Batch(cmd, centerCmd)
		case "left", "h", "ctrl+b", "[":
			cmd = m.HandleButtonPress(SeekBackwardButton)
			m.seekBackward()
			return m, tea.Batch(cmd, centerCmd)
		case "n", "ctrl+s":
			cmd = m.HandleButtonPress(NextButton)
			m.next()
			return m, tea.Batch(cmd, centerCmd)
		case "N", "ctrl+r":
			cmd = m.HandleButtonPress(PreviousButton)
			m.previous()
			return m, tea.Batch(cmd, centerCmd)
		case "j", "ctrl+p":
			m.decrementVolume()
			return m, tea.Batch(cmd, centerCmd)
		case "k", "ctrl+n":
			m.incrementVolume()
			return m, tea.Batch(cmd, centerCmd)
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
