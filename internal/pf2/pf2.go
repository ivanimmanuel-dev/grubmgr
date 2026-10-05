// Package pf2 inspects PFF2 section structure and embedded font names.
package pf2

import (
	"encoding/binary"
	"fmt"
	"strings"
)

func Inspect(b []byte) (string, error) {
	if len(b) < 12 || string(b[:4]) != "FILE" || binary.BigEndian.Uint32(b[4:8]) != 4 || string(b[8:12]) != "PFF2" {
		return "", fmt.Errorf("invalid PFF2 signature")
	}
	name := ""
	chix := false
	data := false
	seen := map[string]bool{}
	for off := 12; off < len(b); {
		if len(b)-off < 8 {
			return "", fmt.Errorf("truncated PF2 section")
		}
		tag := string(b[off : off+4])
		n := uint64(binary.BigEndian.Uint32(b[off+4 : off+8]))
		off += 8
		if seen[tag] {
			return "", fmt.Errorf("duplicate PF2 section %s", tag)
		}
		seen[tag] = true
		if tag == "DATA" && n == 0xffffffff {
			n = uint64(len(b) - off)
		}
		if n > uint64(len(b)-off) {
			return "", fmt.Errorf("PF2 section exceeds file")
		}
		v := b[off : off+int(n)]
		switch tag {
		case "NAME":
			if len(v) > 256 || len(v) == 0 || v[len(v)-1] != 0 {
				return "", fmt.Errorf("invalid PF2 name")
			}
			name = strings.TrimRight(string(v), "\x00")
		case "CHIX":
			if len(v)%9 != 0 {
				return "", fmt.Errorf("invalid PF2 character index")
			}
			chix = true
		case "DATA":
			data = true
		}
		off += int(n)
	}
	if name == "" || !chix || !data {
		return "", fmt.Errorf("PF2 requires NAME, CHIX and DATA sections")
	}
	return name, nil
}
