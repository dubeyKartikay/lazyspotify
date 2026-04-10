package mediacenter

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, m.keys.TogglePanel) {
			m.mediaListOpen = !m.mediaListOpen
			return nil
		}
		if !m.mediaListOpen {
			return nil
		}
	}
	return m.mediaPanel.Update(msg)
}
