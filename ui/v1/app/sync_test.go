package app

import (
	"testing"
	"time"
)

func TestCurrentPositionMsWhenPaused(t *testing.T) {
	m := &Model{}
	m.posAnchorMs = 5000
	m.posAnchorWall = time.Now().Add(-3 * time.Second)
	m.playing = false

	if got := m.currentPositionMs(); got != 5000 {
		t.Fatalf("paused: got %d want 5000", got)
	}
}

func TestCurrentPositionMsAdvancesByWallClock(t *testing.T) {
	m := &Model{}
	m.songInfo.Duration = 200000
	m.posAnchorMs = 1000
	m.posAnchorWall = time.Now().Add(-2 * time.Second)
	m.playing = true

	got := m.currentPositionMs()
	if got < 2900 || got > 3100 {
		t.Fatalf("expected ~3000ms, got %d", got)
	}
}

func TestCurrentPositionMsClampedToDuration(t *testing.T) {
	m := &Model{}
	m.songInfo.Duration = 4000
	m.posAnchorMs = 3000
	m.posAnchorWall = time.Now().Add(-10 * time.Second)
	m.playing = true

	if got := m.currentPositionMs(); got != 4000 {
		t.Fatalf("expected clamp to 4000, got %d", got)
	}
}

func TestSetPositionAnchorResetsWallClock(t *testing.T) {
	m := &Model{}
	old := time.Now().Add(-5 * time.Second)
	m.posAnchorWall = old
	m.setPositionAnchor(12345)

	if m.posAnchorMs != 12345 {
		t.Fatalf("anchor ms not set: %d", m.posAnchorMs)
	}
	if !m.posAnchorWall.After(old) {
		t.Fatalf("expected wall clock to be updated")
	}
}
