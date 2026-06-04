package librespot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

func TestToWebSocketURL(t *testing.T) {
	tests := []struct {
		name      string
		serverURL string
		want      string
	}{
		{name: "http", serverURL: "http://127.0.0.1:4040", want: "ws://127.0.0.1:4040"},
		{name: "https", serverURL: "https://lazyspotify.test", want: "wss://lazyspotify.test"},
		{name: "bare host", serverURL: "127.0.0.1:4040", want: "ws://127.0.0.1:4040"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toWebSocketURL(tt.serverURL); got != tt.want {
				t.Fatalf("toWebSocketURL(%q) = %q, want %q", tt.serverURL, got, tt.want)
			}
		})
	}
}

func TestEventSocketConnectAndReadForwardsSupportedEventsAndIgnoresUnsupported(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events" {
			t.Fatalf("path = %q, want /events", r.URL.Path)
		}

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Fatalf("websocket.Accept() error = %v, want nil", err)
		}
		defer conn.Close(websocket.StatusNormalClosure, "done")

		envelopes := []models.EventEnvelope{
			{
				Type: models.EventTypeMetadata,
				Data: rawJSON(t, `{
					"context_uri": "spotify:playlist:playlist-id",
					"uri": "spotify:track:track-id",
					"name": "Track",
					"artist_names": ["Artist"],
					"album_name": "Album",
					"position": 12000,
					"duration": 180000
				}`),
			},
			{
				Type: "unknown",
				Data: rawJSON(t, `{}`),
			},
			{
				Type: models.EventTypeVolume,
				Data: rawJSON(t, `{"value": 32768, "max": 65535}`),
			},
		}

		for _, envelope := range envelopes {
			if err := wsjson.Write(r.Context(), conn, envelope); err != nil {
				t.Fatalf("wsjson.Write(%s) error = %v, want nil", envelope.Type, err)
			}
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	socket := newEventSocket(server.URL)
	err := socket.connectAndRead()
	if status := websocket.CloseStatus(err); status != websocket.StatusNormalClosure {
		t.Fatalf("connectAndRead() error = %v, close status = %d, want normal closure", err, status)
	}

	first := readBufferedEvent(t, socket)
	if first.Type != models.EventTypeMetadata || first.Metadata == nil {
		t.Fatalf("first event = %#v, want metadata event", first)
	}
	if first.Metadata.URI != "spotify:track:track-id" || first.Metadata.Name != "Track" {
		t.Fatalf("metadata event = %#v, want track-id Track", first.Metadata)
	}

	second := readBufferedEvent(t, socket)
	if second.Type != models.EventTypeVolume || second.Volume == nil {
		t.Fatalf("second event = %#v, want volume event", second)
	}
	if second.Volume.Value != 32768 || second.Volume.Max != 65535 {
		t.Fatalf("volume event = %#v, want 32768/65535", second.Volume)
	}

	select {
	case ev := <-socket.events:
		t.Fatalf("unexpected extra event: %#v", ev)
	default:
	}
}

func rawJSON(t *testing.T, value string) json.RawMessage {
	t.Helper()
	if !json.Valid([]byte(value)) {
		t.Fatalf("invalid test JSON: %s", value)
	}
	return json.RawMessage(value)
}

func readBufferedEvent(t *testing.T, socket *eventSocket) models.PlayerEvent {
	t.Helper()
	select {
	case event := <-socket.events:
		return event
	default:
		t.Fatal("expected buffered websocket event")
	}
	return models.PlayerEvent{}
}
