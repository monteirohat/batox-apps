package overlay

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestStringPadding(t *testing.T) {
	cases := map[string]int{
		"":     8,  // length word + NUL, padded to 4
		"abc":  8,  // 3 bytes + NUL fills one word
		"abcd": 12, // 4 bytes + NUL needs a second word
	}
	for s, want := range cases {
		a := new(args).str(s)
		if len(a.b) != want {
			t.Errorf("str(%q) is %d bytes, want %d", s, len(a.b), want)
		}
		if n := binary.LittleEndian.Uint32(a.b); int(n) != len(s)+1 {
			t.Errorf("str(%q) length word = %d, want %d", s, n, len(s)+1)
		}
	}
}

func TestStringRoundTrip(t *testing.T) {
	a := new(args).str("zwlr_layer_shell_v1").u32(42)
	s, off := readString(a.b, 0)
	if s != "zwlr_layer_shell_v1" {
		t.Fatalf("readString = %q", s)
	}
	if v, _ := readU32(a.b, off); v != 42 {
		t.Errorf("value after the string = %d, want 42", v)
	}
}

func TestEncodeMessageHeader(t *testing.T) {
	msg := encodeMessage(7, 3, new(args).u32(1).u32(2))
	if len(msg) != 16 {
		t.Fatalf("message is %d bytes, want 16", len(msg))
	}
	obj := binary.LittleEndian.Uint32(msg[0:])
	word := binary.LittleEndian.Uint32(msg[4:])
	if obj != 7 || word>>16 != 16 || word&0xffff != 3 {
		t.Errorf("header = obj %d, size %d, op %d", obj, word>>16, word&0xffff)
	}
	if !bytes.Equal(encodeMessage(1, 0, nil)[4:8], []byte{0, 0, 8, 0}) {
		t.Error("an empty message should declare size 8")
	}
}

func TestReadPastEnd(t *testing.T) {
	if v, off := readU32([]byte{1, 2}, 0); v != 0 || off != 2 {
		t.Errorf("short read = %d, %d", v, off)
	}
	if s, _ := readString([]byte{50, 0, 0, 0, 'a'}, 0); s != "" {
		t.Errorf("truncated string = %q, want empty", s)
	}
}
