package lyricsview

import (
	"strings"

	"github.com/rivo/uniseg"
)

// wrapLine word-wraps s to rows of at most width display cells, using
// uniseg's grapheme-cluster width as the single oracle. Words wider than
// width are split on grapheme boundaries.
func wrapLine(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return []string{""}
	}

	var rows []string
	var cur strings.Builder
	curW := 0

	flush := func() {
		if cur.Len() > 0 {
			rows = append(rows, cur.String())
			cur.Reset()
			curW = 0
		}
	}

	for _, word := range strings.Fields(s) {
		ww := uniseg.StringWidth(word)
		if ww <= width {
			sep := 0
			if curW > 0 {
				sep = 1
			}
			if curW+sep+ww > width {
				flush()
				sep = 0
			}
			if sep == 1 {
				cur.WriteByte(' ')
				curW++
			}
			cur.WriteString(word)
			curW += ww
			continue
		}
		// word longer than width: break into grapheme chunks.
		if curW > 0 {
			flush()
		}
		chunks := splitWideWord(word, width)
		for i, c := range chunks {
			if i < len(chunks)-1 {
				rows = append(rows, c)
			} else {
				cur.WriteString(c)
				curW = uniseg.StringWidth(c)
			}
		}
	}
	flush()
	if len(rows) == 0 {
		return []string{""}
	}
	return rows
}

func splitWideWord(word string, width int) []string {
	var chunks []string
	var cur strings.Builder
	curW := 0
	gr := uniseg.NewGraphemes(word)
	for gr.Next() {
		c := gr.Str()
		cw := uniseg.StringWidth(c)
		if cw <= 0 {
			cur.WriteString(c)
			continue
		}
		if curW+cw > width {
			chunks = append(chunks, cur.String())
			cur.Reset()
			curW = 0
		}
		cur.WriteString(c)
		curW += cw
	}
	if cur.Len() > 0 {
		chunks = append(chunks, cur.String())
	}
	return chunks
}

// truncateToWidth returns a string whose display width does not exceed width,
// adding an ellipsis if it had to cut. Uses uniseg as the width oracle.
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if uniseg.StringWidth(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	target := width - 1
	var b strings.Builder
	w := 0
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		c := gr.Str()
		cw := uniseg.StringWidth(c)
		if w+cw > target {
			break
		}
		b.WriteString(c)
		w += cw
	}
	b.WriteString("…")
	return b.String()
}

// padRight pads s with ASCII spaces so its display width equals width.
// Used to make every visual row fill the same number of cells, which keeps
// background fills (when used) consistent. Built on the same width oracle.
func padRight(s string, width int) string {
	w := uniseg.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
