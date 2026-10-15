package auth

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateCodeChallengeUsesS256PKCEEncoding(t *testing.T) {
	// RFC 7636 Appendix B test vector.
	codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if got := generateCodeChallenge(codeVerifier); got != want {
		t.Fatalf("generateCodeChallenge() = %q, want %q", got, want)
	}
}

func TestGenerateRandomStringReturnsRawURLSafeBase64(t *testing.T) {
	got := generateRandomString(32)

	if got == "" {
		t.Fatal("generateRandomString() = empty string, want random value")
	}
	if strings.Contains(got, "=") {
		t.Fatalf("generateRandomString() = %q, want unpadded base64url", got)
	}
	if _, err := base64.RawURLEncoding.DecodeString(got); err != nil {
		t.Fatalf("generateRandomString() produced invalid raw base64url %q: %v", got, err)
	}
}

func TestParsePastedCallback(t *testing.T) {
	config := &AuthConfig{state: "session-state"}
	for _, tt := range []struct {
		name, pasted, want string
	}{
		{"full URL", "http://127.0.0.1:8287/callback?code=auth-code&state=session-state", "auth-code"},
		{"whitespace", " \nhttp://127.0.0.1:8287/callback?code=auth-code&state=session-state\n ", "auth-code"},
		{"encoded code", "http://127.0.0.1:8287/callback?state=session-state&code=code%2Bwith%2Fsymbols%3D", "code+with/symbols="},
		{"empty", "", ""},
		{"bare code", "auth-code", ""},
		{"missing state", "http://127.0.0.1:8287/callback?code=auth-code", ""},
		{"wrong session", "http://127.0.0.1:8287/callback?code=auth-code&state=old-session", ""},
		{"missing code", "http://127.0.0.1:8287/callback?state=session-state", ""},
		{"denied", "http://127.0.0.1:8287/callback?state=session-state&error=access_denied", ""},
		{"malformed query", "http://127.0.0.1:8287/callback?state=session-state&code=%zz", ""},
		{"duplicate state", "http://127.0.0.1:8287/callback?code=auth-code&state=session-state&state=other", ""},
		{"duplicate code", "http://127.0.0.1:8287/callback?code=one&code=two&state=session-state", ""},
		{"wrong URL", "https://accounts.spotify.com/authorize?client_id=client-id", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := config.ParsePastedCallback(tt.pasted)
			if (err != nil) != (tt.want == "") || got != tt.want {
				t.Fatalf("ParsePastedCallback() = %q, %v, want %q", got, err, tt.want)
			}
		})
	}
}
