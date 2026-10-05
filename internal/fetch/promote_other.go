//go:build !windows

package fetch

import "os"

func promote(source, destination string) error { return os.Rename(source, destination) }
