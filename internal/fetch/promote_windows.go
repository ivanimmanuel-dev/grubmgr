//go:build windows

package fetch

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"time"
)

// A scanner can briefly open newly validated files without FILE_SHARE_DELETE.
// Retry only that class of Windows error; a persistent denial still fails.
func promote(source, destination string) error {
	var err error
	for attempt := 0; attempt < 10; attempt++ {
		err = os.Rename(source, destination)
		if err == nil || (!errors.Is(err, windows.ERROR_SHARING_VIOLATION) && !errors.Is(err, windows.ERROR_ACCESS_DENIED)) {
			return err
		}
		if attempt < 9 {
			time.Sleep(50 * time.Millisecond)
		}
	}
	return err
}
