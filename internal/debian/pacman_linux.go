//go:build linux

package debian

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const pacmanOwner = "grubmgr pacman transaction lock v1\n"

// The caller holds operation.lock. Pacman uses exclusive creation, not flock.
// Linking our durable ownership file makes creation atomic while retaining an
// inode identity that lets recover remove only our own lock after SIGKILL.
func acquirePacman(root string) (func(), error) {
	owner := filepath.Join(root, "var/lib/grubmgr/pacman-lock-owner")
	lock := filepath.Join(root, "var/lib/pacman/db.lck")
	for _, directory := range []string{filepath.Dir(owner), filepath.Dir(lock)} {
		if err := secure(directory, true); err != nil {
			return nil, err
		}
	}
	syncDir := func(name string) error {
		d, err := os.Open(filepath.Dir(name))
		if err != nil {
			return err
		}
		defer d.Close()
		return d.Sync()
	}
	owned := func() (os.FileInfo, error) {
		info, err := os.Lstat(owner)
		if err != nil {
			return nil, err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || !info.Mode().IsRegular() || st.Uid != 0 || info.Mode().Perm() != 0600 || st.Nlink > 2 {
			return nil, fmt.Errorf("invalid pacman lock ownership record")
		}
		b, err := os.ReadFile(owner)
		if err != nil || !bytes.Equal(b, []byte(pacmanOwner)) {
			return nil, fmt.Errorf("pacman lock ownership record changed")
		}
		return info, nil
	}
	clean := func() error {
		info, err := owned()
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		other, err := os.Lstat(lock)
		if err == nil {
			if !os.SameFile(info, other) {
				return fmt.Errorf("pacman lock belongs to another operation; leaving it untouched")
			}
			if err = os.Remove(lock); err != nil {
				return err
			}
			if err = syncDir(lock); err != nil {
				return err
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		if err = os.Remove(owner); err != nil {
			return err
		}
		return syncDir(owner)
	}
	if err := clean(); err != nil {
		return nil, err
	}
	if _, err := os.Lstat(lock); !os.IsNotExist(err) {
		return nil, fmt.Errorf("pacman database is locked; no lock was removed")
	}
	f, err := os.OpenFile(owner, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if _, err = f.WriteString(pacmanOwner); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = syncDir(owner)
	}
	if err == nil {
		err = os.Link(owner, lock)
	}
	if err != nil {
		_ = clean()
		return nil, fmt.Errorf("cannot lock pacman database: %w", err)
	}
	if err = syncDir(lock); err != nil {
		_ = clean()
		return nil, err
	}
	return func() { _ = clean() }, nil
}
