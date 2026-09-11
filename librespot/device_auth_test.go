package librespot

import (
	"context"
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
