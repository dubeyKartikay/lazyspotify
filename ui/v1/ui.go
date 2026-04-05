package v1

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
)

func newModel() Model {
	return Model{
		cassette: NewCassette(),
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
		return nil
	}
	return tea.Batch(cmd, DoTickSpokes())
}

func (m *Model) View() tea.View {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	if m.authModel != nil && m.authModel.needsAuth {
		return m.authModel.View()
	}
	cassette := m.cassette
	v := cassette.View()
	return tea.NewView(v + "\n" + helpStyle.Render("Press q to quit"))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)
		return m, nil
	case NextFrameMsg:
		logger.Log.Debug().Msg("next frame")
		m.spoke.NextFrame()
		return m, DoTickSpokes()
	}
	if m.authModel != nil && m.authModel.needsAuth {
		newM, cmd := m.authModel.Update(msg)
		m.authModel = newM.(*AuthModel)
		return m, cmd
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case " ", "p":
			m.playPause()
			return m, cmd
		case "right", "l", "ctrl+f", "]":
			m.seekForward()
			return m, cmd
		case "left", "h", "ctrl+b", "[":
			m.seekBackward()
			return m, cmd
		case "n", "ctrl+s":
			m.next()
			return m, cmd
		case "N", "ctrl+r":
			m.previous()
			return m, cmd
		case "j", "down", "ctrl+p":
			m.decrementVolume()
			return m, cmd
		case "k", "up", "ctrl+n":
			m.incrementVolume()
			return m, cmd
		}
	}
	return m, nil
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
