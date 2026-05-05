package lyricsexport

import (
	"encoding/json"
	"net"
	"os"
	"sync"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
)

// Snapshot is one NDJSON record for external consumers (polybar, waybar, etc.).
type Snapshot struct {
	V           int    `json:"v"`
	TrackURI    string `json:"track_uri"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	PositionMs  int    `json:"position_ms"`
	DurationMs  int    `json:"duration_ms"`
	Playing     bool   `json:"playing"`
	LineIndex   int    `json:"line_index"`
	LineText    string `json:"line_text"`
	SyncType    string `json:"sync_type"`
	LyricsError string `json:"lyrics_error,omitempty"`
}

// Broadcaster listens on a Unix domain socket and writes newline-delimited JSON
// snapshots to every connected client whenever Publish is called.
type Broadcaster struct {
	path  string
	ln    net.Listener
	mu    sync.Mutex
	conns []net.Conn
}

// Start removes any stale socket file, listens on path, and accepts clients in
// the background. path must be non-empty.
func Start(path string) (*Broadcaster, error) {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	b := &Broadcaster{path: path, ln: ln}
	go b.acceptLoop()
	logger.Log.Info().Str("path", path).Msg("lyrics export socket listening")
	return b, nil
}

func (b *Broadcaster) acceptLoop() {
	for {
		c, err := b.ln.Accept()
		if err != nil {
			return
		}
		b.mu.Lock()
		b.conns = append(b.conns, c)
		b.mu.Unlock()
	}
}

// Publish marshals snap as one JSON line and writes it to all active connections.
func (b *Broadcaster) Publish(snap Snapshot) {
	if b == nil || b.ln == nil {
		return
	}
	snap.V = 1
	payload, err := json.Marshal(snap)
	if err != nil {
		return
	}
	payload = append(payload, '\n')

	deadline := time.Now().Add(2 * time.Second)

	b.mu.Lock()
	defer b.mu.Unlock()

	alive := b.conns[:0]
	for _, c := range b.conns {
		_ = c.SetWriteDeadline(deadline)
		if _, err := c.Write(payload); err != nil {
			_ = c.Close()
			continue
		}
		alive = append(alive, c)
	}
	b.conns = alive
}

// Close stops listening and disconnects all clients.
func (b *Broadcaster) Close() {
	if b == nil {
		return
	}
	if b.ln != nil {
		_ = b.ln.Close()
		b.ln = nil
	}
	b.mu.Lock()
	for _, c := range b.conns {
		_ = c.Close()
	}
	b.conns = nil
	b.mu.Unlock()
	if b.path != "" {
		_ = os.Remove(b.path)
	}
}
