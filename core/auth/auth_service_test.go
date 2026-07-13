package auth

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

type authRoundTripper func(*http.Request) (*http.Response, error)

func (f authRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPastedCallbackExchangesOriginalPKCEAndDeliversToken(t *testing.T) {
	const redirectURI = "http://127.0.0.1:8287/callback"
	service := NewAuthService(redirectURI)
	service.authConfig.state = "session-state"
	service.authConfig.codeVerifier = "original-verifier"
	allowExchange := false
	requests := 0
	client := &http.Client{Transport: authRoundTripper(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.URL.String() != "https://accounts.spotify.com/api/token" {
			t.Fatalf("unexpected request URL: %s", r.URL)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		for field, want := range map[string]string{
			"grant_type": "authorization_code", "code": "auth-code",
			"code_verifier": "original-verifier", "redirect_uri": redirectURI,
		} {
			if got := r.Form.Get(field); got != want {
				t.Errorf("%s = %q, want %q", field, got, want)
			}
		}
		status, body := http.StatusBadRequest, `{"error":"invalid_grant"}`
		if allowExchange {
			status, body = http.StatusOK, `{"access_token":"web-token","refresh_token":"refresh-token","token_type":"Bearer","expires_in":3600}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)

	if _, err := service.ExchangePastedCallback(ctx, redirectURI+"?code=auth-code&state=wrong-session"); err == nil || requests != 0 {
		t.Fatal("mismatched state must be rejected before making a token request")
	}
	callback := redirectURI + "?code=auth-code&state=session-state"
	if _, err := service.ExchangePastedCallback(ctx, callback); err == nil {
		t.Fatal("failed token exchange should remain retryable")
	}
	select {
	case <-service.GetTokenChannel():
		t.Fatal("failed exchange completed the login")
	default:
	}
	allowExchange = true
	token, err := service.ExchangePastedCallback(ctx, callback)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case delivered := <-service.GetTokenChannel():
		if delivered != token || delivered.AccessToken != "web-token" || delivered.RefreshToken != "refresh-token" {
			t.Fatalf("unexpected token delivered: %+v", delivered)
		}
	default:
		t.Fatal("pasted callback did not complete the waiting login flow")
	}
}

func TestConcurrentTokenDeliveryCompletesLoginOnce(t *testing.T) {
	service := NewAuthService("http://127.0.0.1:8287/callback")
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			service.sendToken(&oauth2.Token{AccessToken: "token"})
		}()
	}
	wg.Wait()
	<-service.GetTokenChannel()
	service.sendToken(&oauth2.Token{AccessToken: "late-token"})
	select {
	case <-service.GetTokenChannel():
		t.Fatal("a late callback queued another token after login completed")
	default:
	}
}
