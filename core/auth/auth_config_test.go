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
