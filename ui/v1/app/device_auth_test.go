package app

import (
	"strings"
	"testing"
	"time"

	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

func TestDeviceAuthScreenAndApproval(t *testing.T) {
	m := NewModel()
	m.width, m.height = 100, 30
	m.handleSystemMessages(deviceAuthMsg{code: &models.DeviceAuth{URL: "https://spotify.com/pair", Code: "ABC123", ExpiresAt: time.Now().Add(5 * time.Minute)}})
	screen := m.deviceAuthView()
	for _, s := range []string{"https://spotify.com/pair", "ABC123", "copy pairing link"} {
		if !strings.Contains(screen, s) {
			t.Fatalf("pairing screen missing %q", s)
		}
	}
	m.handleSystemMessages(playerReadyMsg{})
	if m.deviceAuth != nil || !m.playerReady {
		t.Fatal("pairing screen did not close on approval")
	}
}
