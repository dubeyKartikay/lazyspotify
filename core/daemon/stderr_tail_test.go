package daemon

import (
	"bytes"
	"testing"
)

func TestDaemonErrorOutputKeepsNewestBytesAcrossWrites(t *testing.T) {
	first := bytes.Repeat([]byte("a"), 3000)
	second := bytes.Repeat([]byte("b"), 2000)
	output := &daemonErrorOutput{}

	if n, err := output.Write(first); err != nil || n != len(first) {
		t.Fatalf("Write(first) = %d, %v", n, err)
	}
	if n, err := output.Write(second); err != nil || n != len(second) {
		t.Fatalf("Write(second) = %d, %v", n, err)
	}

	// 5000 bytes were written, so the newest 4096 must remain: the tail of
	// the first write followed by all of the second.
	want := append(bytes.Repeat([]byte("a"), 4096-len(second)), second...)
	if !bytes.Equal(output.tail, want) {
		t.Fatalf("tail has %d bytes (%d a, %d b), want the newest 4096 (%d a, %d b)",
			len(output.tail), bytes.Count(output.tail, []byte("a")), bytes.Count(output.tail, []byte("b")),
			4096-len(second), len(second))
	}
}
