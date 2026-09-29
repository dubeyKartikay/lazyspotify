package auth

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWritePairingRedirectSendsBrowserToPairingPage(t *testing.T) {
	recorder := httptest.NewRecorder()

	writePairingRedirect(recorder, "https://spotify.com/pair?code=ABC123", "ABC123")

	body := recorder.Body.String()
	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", got)
	}
	for _, want := range []string{
		`http-equiv="refresh" content="0; url=https://spotify.com/pair?code=ABC123"`,
		`href="https://spotify.com/pair?code=ABC123"`,
		"<strong>ABC123</strong>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("redirect page missing %q", want)
		}
	}
}

func TestWritePairingRedirectEscapesProvidedValues(t *testing.T) {
	recorder := httptest.NewRecorder()

	writePairingRedirect(recorder, `https://example.com/pair?code="><script>`, "<script>")

	body := recorder.Body.String()
	if strings.Contains(body, "<script>") {
		t.Fatal("redirect page did not escape provided values")
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatal("redirect page should render escaped values")
	}
}

func TestWriteSuccessFallsBackToPlainMessageWithoutPairing(t *testing.T) {
	service := &AuthService{}
	recorder := httptest.NewRecorder()

	service.writeSuccess(recorder)

	if body := recorder.Body.String(); !strings.Contains(body, "Authentication successful. You can close this window.") {
		t.Fatalf("writeSuccess() body = %q, want plain success message", body)
	}
}

func TestWriteSuccessUsesPairingRedirectWhenDaemonHasPendingCode(t *testing.T) {
	service := &AuthService{}
	service.SetPostAuthRedirect(func() (string, string, bool) {
		return "https://spotify.com/pair?code=ABC123", "ABC123", true
	})
	recorder := httptest.NewRecorder()

	service.writeSuccess(recorder)

	body := recorder.Body.String()
	if !strings.Contains(body, "https://spotify.com/pair?code=ABC123") {
		t.Fatalf("writeSuccess() body = %q, want pairing redirect", body)
	}
	if strings.Contains(body, "You can close this window") {
		t.Fatal("writeSuccess() should not tell the user to close the window while pairing is pending")
	}
}

func TestWriteSuccessIgnoresPairingWithoutPendingCode(t *testing.T) {
	service := &AuthService{}
	service.SetPostAuthRedirect(func() (string, string, bool) {
		return "", "", false
	})
	recorder := httptest.NewRecorder()

	service.writeSuccess(recorder)

	if body := recorder.Body.String(); !strings.Contains(body, "You can close this window") {
		t.Fatalf("writeSuccess() body = %q, want plain success message", body)
	}
}
