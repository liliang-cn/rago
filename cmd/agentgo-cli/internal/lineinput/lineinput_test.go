package lineinput

import (
	"bytes"
	"testing"
)

func TestDecodeRuneFromReaderHandlesChineseUTF8(t *testing.T) {
	r, n := DecodeRuneFromReader(bytes.NewBufferString("\xbd\xa0"), 0xe4)
	if r != '你' {
		t.Fatalf("expected 你, got %q", r)
	}
	if n != 3 {
		t.Fatalf("expected 3-byte rune, got %d", n)
	}
}
