//go:build linux

package debian

import (
	"context"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// terminalAgent registers pkttyagent to handle authentication on the controlling
// terminal, including with pkexec versions whose built-in agent fails.
func terminalAgent(ctx context.Context) (func(), error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return func() {}, nil // A desktop agent may handle a non-terminal caller.
	}
	if err = secure("/usr/bin/pkttyagent", false); err != nil {
		tty.Close()
		return nil, fmt.Errorf("terminal authentication agent unavailable: %w", err)
	}
	terminal, err := unix.IoctlGetTermios(int(tty.Fd()), unix.TCGETS)
	if err != nil {
		tty.Close()
		return nil, err
	}
	stat, err := os.ReadFile("/proc/self/stat")
	fields := strings.Fields(string(stat)[strings.LastIndex(string(stat), ")")+1:])
	if err != nil || len(fields) < 20 {
		tty.Close()
		return nil, fmt.Errorf("cannot identify authentication subject")
	}
	start, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil || start == 0 {
		tty.Close()
		return nil, fmt.Errorf("invalid authentication subject start time")
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		tty.Close()
		return nil, err
	}
	subject := strconv.Itoa(os.Getpid()) + "," + strconv.FormatUint(start, 10)
	cmd := exec.CommandContext(ctx, "/usr/bin/pkttyagent", "--fallback", "--process", subject, "--notify-fd", "3")
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "TERM=dumb"}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, tty
	cmd.ExtraFiles = []*os.File{writer}
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
	if err = cmd.Start(); err != nil {
		reader.Close()
		writer.Close()
		tty.Close()
		return nil, err
	}
	writer.Close()
	done := make(chan struct{})
	var waitErr error
	go func() { waitErr = cmd.Wait(); close(done) }()
	stop := func() {
		_ = cmd.Process.Kill()
		<-done
		reader.Close()
		_ = unix.IoctlSetTermios(int(tty.Fd()), unix.TCSETS, terminal)
		tty.Close()
	}
	ready := make(chan error, 1)
	go func() { _, err := io.Copy(io.Discard, reader); ready <- err }()
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case err = <-ready:
		// pkttyagent closes notify-fd after registering with Polkit.
		select {
		case <-done:
			err = fmt.Errorf("terminal authentication agent exited: %v", waitErr)
		default:
		}
	case <-done:
		err = fmt.Errorf("terminal authentication agent exited: %v", waitErr)
	case <-timer.C:
		err = fmt.Errorf("terminal authentication agent registration timed out")
	case <-ctx.Done():
		err = ctx.Err()
	}
	if err != nil {
		stop()
		return nil, err
	}
	return stop, nil
}
