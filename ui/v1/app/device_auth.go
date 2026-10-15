package app

import (
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

type deviceAuthMsg struct{ code *models.DeviceAuth }
type deviceAuthCopiedMsg struct{ err error }

func (m *Model) waitForDeviceAuth() tea.Cmd {
	if m.player == nil {
		return nil
	}
	codes := m.player.AuthCodes()
	if codes == nil {
		return nil
	}
	return func() tea.Msg {
		code, ok := <-codes
		if !ok {
			return nil
		}
		return deviceAuthMsg{code: code}
	}
}

func (m *Model) handleDeviceAuthInput(msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "c" {
		url := m.deviceAuth.URL
		return func() tea.Msg { return deviceAuthCopiedMsg{err: utils.CopyToClipboard(url)} }
	}
	return nil
}

func (m *Model) deviceAuthView() string {
	auth := m.deviceAuth
	remaining := time.Until(auth.ExpiresAt).Round(time.Second)
	expiry := fmt.Sprintf("Code expires in %s", remaining)
	if remaining <= 0 {
		expiry = "Code expired; waiting for a new code"
	}
	hint := "c: copy pairing link • q: quit"
	if m.deviceAuthHint != "" {
		hint = m.deviceAuthHint + "\n" + hint
	}
	width := max(20, min(80, m.width-4))
	content := lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Bold(true).Render("Sign in to Spotify"),
		"",
		"Open this link on your phone or computer:",
		lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(auth.URL),
		"",
		"Enter this code if prompted:",
		lipgloss.NewStyle().Bold(true).Render(auth.Code),
		"",
		expiry,
		"Lazyspotify will continue after you approve.",
		"",
		hint,
	)
	return lipgloss.NewStyle().Width(m.width).Height(m.height).Align(lipgloss.Center, lipgloss.Center).Render(content)
}
