//go:build windows

package fetch

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPromoteAfterTemporarySharingLock(t *testing.T) {
	root := t.TempDir()
	source, destination := filepath.Join(root, "source"), filepath.Join(root, "destination")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(source, "theme.txt")
	if err := os.WriteFile(file, []byte("theme data"), 0600); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(file)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { time.Sleep(100 * time.Millisecond); _ = windows.CloseHandle(handle); close(done) }()
	err = promote(source, destination)
	<-done
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(destination, "theme.txt"))
	if err != nil || string(data) != "theme data" {
		t.Fatal(string(data), err)
	}
}
