// Package backend generates configuration for synthetic filesystem fixtures.
package backend

import (
	"bytes"
	"fmt"
	"grubmgr/internal/system"
	"strings"
)

const Defaults = "etc/default/grub"
const Config = "boot/grub/grub.cfg"

func Settings(before []byte, theme string) ([]byte, error) {
	_, simple := system.Theme(before)
	if !simple {
		return nil, fmt.Errorf("complex theme assignment refused")
	}
	lines := strings.Split(strings.TrimSuffix(string(before), "\n"), "\n")
	out := []string{}
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "GRUB_THEME=") {
			continue
		}
		out = append(out, line)
	}
	if theme != "" {
		out = append(out, `GRUB_THEME="`+theme+`"`)
	}
	return []byte(strings.Join(out, "\n") + "\n"), nil
}

const start = "# grubmgr fixture theme begin"
const end = "# grubmgr fixture theme end"

// Candidate updates the fixture theme block and preserves surrounding bytes.
func Candidate(current []byte, theme string) ([]byte, error) {
	b := current
	a := bytes.Index(b, []byte(start))
	if a >= 0 {
		z := bytes.Index(b[a:], []byte(end+"\n"))
		if z < 0 || bytes.Count(b, []byte(start)) != 1 || bytes.Count(b, []byte(end)) != 1 {
			return nil, fmt.Errorf("malformed synthetic theme block")
		}
		b = append(append([]byte{}, b[:a]...), b[a+z+len(end)+1:]...)
	}
	if len(b) > 0 && b[len(b)-1] != '\n' {
		return nil, fmt.Errorf("fixture configuration must end with newline")
	}
	if theme != "" {
		b = append(b, []byte(start+"\nset theme=\""+theme+"\"\n"+end+"\n")...)
	}
	return b, nil
}
