package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/dubeyKartikay/lazyspotify/core/utils"
)

type AuthConfig struct {
	codeVerifier  string
	codeChallenge string
	state         string
	clientID      string
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		panic("rand.Read failed")
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(codeVerifier string) string {
	h := sha256.New()
	h.Write([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func NewAuthConfig() *AuthConfig {
	codeVerifier := generateRandomString(32)
	state := generateRandomString(32)
	codeChallenge := generateCodeChallenge(codeVerifier)
	clientID := utils.GetConfig().SpotifyClientID()

	return &AuthConfig{
		codeVerifier:  codeVerifier,
		codeChallenge: codeChallenge,
		state:         state,
		clientID:      clientID,
	}
}

// ParsePastedCallback extracts the authorization code from a value the user
// pasted back after approving the login. The full URL preserves the state
// parameter, so remote logins receive the same validation as local callbacks.
func (c *AuthConfig) ParsePastedCallback(pasted string) (string, error) {
	trimmed := strings.TrimSpace(pasted)
	if trimmed == "" {
		return "", errors.New("empty callback")
	}
	callbackURL, err := url.Parse(trimmed)
	if err != nil {
		return "", errors.New("invalid callback URL; paste the full address bar URL after approving")
	}
	if (callbackURL.Scheme != "http" && callbackURL.Scheme != "https") || callbackURL.Host == "" {
		return "", errors.New("paste the full address bar URL after approving")
	}
	values, err := url.ParseQuery(callbackURL.RawQuery)
	if err != nil {
		return "", errors.New("invalid callback query; paste the full address bar URL again")
	}
	if len(values["state"]) != 1 || values.Get("state") != c.state {
		return "", errors.New("callback is from a different login session; approve again using the URL above")
	}
	if e := values.Get("error"); e != "" {
		return "", fmt.Errorf("spotify rejected the login: %s", e)
	}
	code := values.Get("code")
	if code == "" || len(values["code"]) != 1 {
		return "", errors.New("no code found: paste the full address bar url after approving")
	}
	return code, nil
}
