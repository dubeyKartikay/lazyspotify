package lyricsview

import (
	"testing"

	"github.com/rivo/uniseg"
)

func TestWrapLineFitsWidth(t *testing.T) {
	cases := []struct {
		name  string
		input string
		width int
	}{
		{"ascii", "the quick brown fox jumps over the lazy dog", 12},
		{"cjk", "あいうえお かきくけこ さしすせそ たちつてと", 8},
		{"devanagari", "नमस्ते दुनिया यह एक परीक्षण पंक्ति है", 14},
		{"emoji_zwj", "family 👨‍👩‍👧 dancing 💃 here", 10},
		{"long_word", "supercalifragilisticexpialidocious", 8},
		{"mixed", "hello 世界 emoji 🎵 and more text", 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows := wrapLine(tc.input, tc.width)
			if len(rows) == 0 {
				t.Fatalf("no rows produced")
			}
			for i, r := range rows {
				if w := uniseg.StringWidth(r); w > tc.width {
					t.Errorf("row %d width=%d exceeds width=%d: %q", i, w, tc.width, r)
				}
			}
		})
	}
}

func TestWrapLineEmpty(t *testing.T) {
	rows := wrapLine("", 10)
	if len(rows) != 1 || rows[0] != "" {
		t.Fatalf("expected single empty row, got %v", rows)
	}
}

func TestPadRightDisplayWidth(t *testing.T) {
	got := padRight("世界", 6)
	if w := uniseg.StringWidth(got); w != 6 {
		t.Fatalf("padRight width=%d want 6: %q", w, got)
	}
}

func TestTruncateToWidth(t *testing.T) {
	got := truncateToWidth("hello world", 7)
	if w := uniseg.StringWidth(got); w > 7 {
		t.Fatalf("truncate width=%d > 7: %q", w, got)
	}
	got2 := truncateToWidth("short", 20)
	if got2 != "short" {
		t.Fatalf("expected unchanged, got %q", got2)
	}
}
