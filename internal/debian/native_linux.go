//go:build linux

package debian

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"grubmgr/internal/fsx"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func secure(name string, directory bool) error {
	if err := fsx.NoLinks(name); err != nil {
		return err
	}
	for current := name; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		st, ok := info.Sys().(*syscall.Stat_t)
		if !ok || st.Uid != 0 || info.Mode().Perm()&0022 != 0 {
			return fmt.Errorf("not root-owned and protected: %s", current)
		}
		if current == name && ((directory && !info.IsDir()) || (!directory && !info.Mode().IsRegular())) {
			return fmt.Errorf("unexpected file type: %s", current)
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	return nil
}

// atomicReplace writes on the destination filesystem, preserves mode/owner, and
// syncs both the file and its directory. First-target files have no extra xattrs.
func atomicReplace(r *os.Root, name string, data []byte) error {
	if n, err := unix.Listxattr("/"+name, nil); err != nil && err != unix.ENOTSUP {
		return err
	} else if n > 0 {
		return fmt.Errorf("extended attributes require a separate backend: %s", name)
	}
	info, err := r.Lstat(name)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("invalid replacement target")
	}
	st := info.Sys().(*syscall.Stat_t)
	if st.Uid != 0 || st.Nlink != 1 || info.Mode().Perm()&0022 != 0 {
		return fmt.Errorf("untrusted replacement target")
	}
	tmp := name + ".grubmgr-new"
	f, err := r.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer r.Remove(tmp)
	if err = f.Chown(int(st.Uid), int(st.Gid)); err == nil {
		err = f.Chmod(info.Mode().Perm())
	}
	if err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = r.Rename(tmp, name); err != nil {
		return err
	}
	d, err := r.Open(filepath.Dir(name))
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func acquire(root string) (func(), error) {
	var held []*os.File
	release := func() {
		for i := len(held) - 1; i >= 0; i-- {
			held[i].Close()
		}
	}
	for _, item := range []struct {
		name  string
		flock bool
	}{{"var/lib/grubmgr/operation.lock", true}, {"var/lib/dpkg/lock-frontend", false}, {"var/lib/dpkg/lock", false}} {
		name := filepath.Join(root, item.name)
		if err := secure(filepath.Dir(name), true); err != nil {
			release()
			return nil, err
		}
		f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
		if err != nil {
			release()
			return nil, err
		}
		held = append(held, f)
		if err = secure(name, false); err != nil {
			release()
			return nil, err
		}
		if item.flock {
			err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		} else {
			err = unix.FcntlFlock(f.Fd(), unix.F_SETLK, &unix.Flock_t{Type: unix.F_WRLCK, Whence: io.SeekStart})
		}
		if err != nil {
			release()
			return nil, fmt.Errorf("grubmgr or a package update is active: %w", err)
		}
	}
	return release, nil
}

type boundedOutput struct{ text strings.Builder }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.text.Len()+len(p) > 1<<20 {
		return 0, fmt.Errorf("generator output limit")
	}
	return b.text.Write(p)
}

func runTool(executable string, args ...string) error {
	if executable != "/usr/sbin/grub-mkconfig" && executable != "/usr/bin/grub-script-check" {
		return fmt.Errorf("unapproved executable")
	}
	if err := secure(executable, false); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C", "LC_ALL=C", "HOME=/root"}
	cmd.Dir = "/"
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 2 * time.Second
	var out boundedOutput
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s: %w: %s", executable, err, out.text.String())
	}
	return nil
}

func helperIdentity() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("helper requires Polkit authorization")
	}
	uid, err := strconv.ParseUint(os.Getenv("PKEXEC_UID"), 10, 32)
	if err != nil || uid == 0 {
		return fmt.Errorf("invoke this helper through pkexec from an ordinary user")
	}
	return nil
}

func syncTree(root string) error {
	var directories []string
	sync := func(name string) error {
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		return f.Sync()
	}
	err := filepath.WalkDir(root, func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			directories = append(directories, name)
			return nil
		}
		return sync(name)
	})
	if err != nil {
		return err
	}
	for i := len(directories) - 1; i >= 0; i-- {
		if err = sync(directories[i]); err != nil {
			return err
		}
	}
	return nil
}
