package pf2

import (
	"encoding/binary"
	"testing"
)

func section(tag string, b []byte) []byte {
	out := append([]byte(tag), 0, 0, 0, 0)
	binary.BigEndian.PutUint32(out[4:], uint32(len(b)))
	return append(out, b...)
}
func TestInspect(t *testing.T) {
	b := section("FILE", []byte("PFF2"))
	b = append(b, section("NAME", []byte("Fixture Regular 16\x00"))...)
	b = append(b, section("CHIX", nil)...)
	b = append(b, section("DATA", nil)...)
	name, e := Inspect(b)
	if e != nil || name != "Fixture Regular 16" {
		t.Fatal(name, e)
	}
	for _, bad := range [][]byte{nil, b[:10], b[:len(b)-1], append(append([]byte{}, b...), section("NAME", []byte("dup\x00"))...)} {
		if _, e = Inspect(bad); e == nil {
			t.Fatal("accepted malformed PF2")
		}
	}
}
func FuzzInspect(f *testing.F) {
	f.Add([]byte("FILE\x00\x00\x00\x04PFF2"))
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) < 1<<20 {
			_, _ = Inspect(b)
		}
	})
}
