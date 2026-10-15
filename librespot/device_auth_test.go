package librespot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

func TestGetAuthCode(t *testing.T) {
	expiry := time.Now().UTC().Add(10 * time.Minute).Truncate(time.Second)
	for _, tt := range []struct {
		name     string
		status   int
		body     string
		wantCode bool
		wantErr  bool
	}{
		{"pending", 200, `{"url":"https://spotify.com/pair","code":"TEST","expires_at":"` + expiry.Format(time.RFC3339) + `"}`, true, false},
		{"stored login", 204, "", false, false},
		{"old daemon", 404, "", false, true},
		{"old root fallback", 200, `{"playback_ready":false}`, false, true},
		{"invalid json", 200, "{", false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, closeServer := newTestLibrespotClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/auth/code" {
					t.Errorf("path = %s", r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer closeServer()
			got, err := c.GetAuthCode(context.Background())
			if (err != nil) != tt.wantErr || (got != nil) != tt.wantCode {
				t.Fatalf("code = %+v, error = %v", got, err)
			}
			if got != nil && (got.Code != "TEST" || !got.ExpiresAt.Equal(expiry)) {
				t.Fatalf("code = %+v", got)
			}
		})
	}
}

func TestPairingExtendsReadinessDeadline(t *testing.T) {
	initial := time.Now().Add(3 * time.Minute)
	code := &models.DeviceAuth{ExpiresAt: initial.Add(7 * time.Minute)}
	got := readinessDeadline(initial, code)
	if !got.Equal(code.ExpiresAt.Add(30 * time.Second)) {
		t.Fatal("valid device code would time out early")
	}
	if !readinessDeadline(initial, nil).Equal(initial) {
		t.Fatal("no pending pairing should preserve deadline")
	}
}

func TestReadinessPublishesCodeAndCompletesAfterApproval(t *testing.T) {
	approved := make(chan struct{})
	c, closeServer := newTestLibrespotClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ready := false
		select {
		case <-approved:
			ready = true
		default:
		}
		if r.URL.Path == "/" {
			if !ready {
				t.Error("health requested while device approval is pending")
			}
			_ = json.NewEncoder(w).Encode(models.HealthResponse{PlaybackReady: ready})
			return
		}
		if ready {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		_ = json.NewEncoder(w).Encode(models.DeviceAuth{URL: "https://spotify.com/pair", Code: "TEST", ExpiresAt: time.Now().Add(10 * time.Minute)})
	}))
	defer closeServer()
	l := &Librespot{Client: c, Ready: make(chan error, 1), AuthCodes: make(chan *models.DeviceAuth, 1)}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go notifyWhenReady(ctx, l)
	select {
	case code := <-l.AuthCodes:
		if code == nil || code.Code != "TEST" {
			t.Fatalf("code = %+v", code)
		}
	case <-ctx.Done():
		t.Fatal("no pairing code")
	}
	close(approved)
	select {
	case err := <-l.Ready:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("not ready after approval")
	}
}

func TestReadinessCancellation(t *testing.T) {
	c, closeServer := newTestLibrespotClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	defer closeServer()
	l := &Librespot{Client: c, Ready: make(chan error, 1), AuthCodes: make(chan *models.DeviceAuth, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	go notifyWhenReady(ctx, l)
	select {
	case err := <-l.Ready:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("readiness did not stop")
	}
}
