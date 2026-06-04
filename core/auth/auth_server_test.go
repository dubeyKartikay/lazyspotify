package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAuthServerAddressesComeFromConfiguredFields(t *testing.T) {
	server := &AuthServer{
		host:             "127.0.0.1",
		port:             8287,
		redirectEndpoint: "/callback",
		timeout:          2 * time.Second,
	}

	if got := server.GetAuthServerAddress(); got != "127.0.0.1:8287" {
		t.Fatalf("GetAuthServerAddress() = %q, want 127.0.0.1:8287", got)
	}
	if got := server.GetOauthRedirectURI(); got != "http://127.0.0.1:8287/callback" {
		t.Fatalf("GetOauthRedirectURI() = %q, want http://127.0.0.1:8287/callback", got)
	}
}

func TestRegisterRoutesInvokesCallbackOnlyForRedirectEndpoint(t *testing.T) {
	var callbackHits int
	mux := http.NewServeMux()
	registerRoutes(mux, "/callback", func(w http.ResponseWriter, r *http.Request) {
		callbackHits++
		w.WriteHeader(http.StatusCreated)
	})

	callbackReq := httptest.NewRequest(http.MethodGet, "/callback?code=abc&state=xyz", nil)
	callbackResp := httptest.NewRecorder()
	mux.ServeHTTP(callbackResp, callbackReq)

	if callbackResp.Code != http.StatusCreated {
		t.Fatalf("callback status = %d, want %d", callbackResp.Code, http.StatusCreated)
	}
	if callbackHits != 1 {
		t.Fatalf("callback hits = %d, want 1", callbackHits)
	}

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootResp := httptest.NewRecorder()
	mux.ServeHTTP(rootResp, rootReq)

	if rootResp.Code != http.StatusOK {
		t.Fatalf("root status = %d, want %d", rootResp.Code, http.StatusOK)
	}
	if callbackHits != 1 {
		t.Fatalf("callback hits after root request = %d, want 1", callbackHits)
	}
}

func TestShutdownIsNoopWhenServerWasNotStarted(t *testing.T) {
	server := &AuthServer{timeout: time.Nanosecond}

	if err := server.Shutdown(); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}
}
