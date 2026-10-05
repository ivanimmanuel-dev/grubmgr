//go:build !windows

package fsx

import (
	"fmt"
	"os"
	"syscall"
)

func SingleLink(f *os.File) error {
	i, e := f.Stat()
	if e != nil {
		return e
	}
	s, ok := i.Sys().(*syscall.Stat_t)
	if !ok || s.Nlink != 1 {
		return fmt.Errorf("multiply linked or unsupported file refused")
	}
	return nil
}
