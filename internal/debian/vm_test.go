//go:build linux && grubmgr_vmtest

package debian

import (
	"bufio"
	"bytes"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/planner"
	"grubmgr/internal/state"
	"grubmgr/internal/transaction"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestVMPackageLock(t *testing.T) {
	q, err := Inspect()
	if err != nil || q.Status != "SUPPORTED WITH WARNINGS" || os.Geteuid() != 0 {
		t.Fatal("disposable VM required", q, err)
	}
	unlock, err := acquire("/")
	if err != nil {
		t.Fatal(err)
	}
	if other, err := acquire("/"); err == nil {
		other()
		unlock()
		t.Fatal("concurrent manager accepted")
	}
	unlock()
	if q.profile.PackageManager == "pacman" {
		name := "/var/lib/pacman/db.lck"
		f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.WriteString("foreign package operation\n")
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(name)
		if other, err := acquire("/"); err == nil {
			other()
			t.Fatal("foreign pacman lock accepted")
		}
		data, err := os.ReadFile(name)
		if err != nil || string(data) != "foreign package operation\n" {
			t.Fatal("foreign lock changed", err)
		}
	} else {
		cmd := exec.Command("python3", "-c", "import fcntl,time; f=open('/var/lib/dpkg/lock','r+'); fcntl.lockf(f,fcntl.LOCK_EX); print('ready',flush=True); time.sleep(60)")
		pipe, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err = cmd.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
		line, err := bufio.NewReader(pipe).ReadString('\n')
		if err != nil || line != "ready\n" {
			t.Fatal(line, err)
		}
		if other, err := acquire("/"); err == nil {
			other()
			t.Fatal("dpkg lock accepted")
		}
	}
}

func TestVMEFIAutomount(t *testing.T) {
	q, err := Inspect()
	if err != nil || q.Status != "SUPPORTED WITH WARNINGS" || os.Geteuid() != 0 {
		t.Fatal("disposable VM required", q, err)
	}
	if q.profile.ID != "arch" {
		t.Skip("the Arch image has an EFI automount")
	}
	if out, err := exec.Command("/usr/bin/umount", "/efi").CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	if q, err = Inspect(); err != nil || q.Status != "SUPPORTED WITH WARNINGS" {
		t.Fatal("idle EFI automount was not detected", q, err)
	}
}

// vmPhaseHook interrupts the test process at a selected journal boundary.
func vmPhaseHook(phase string) {
	if os.Getenv("GRUBMGR_VM_KILL_PHASE") == phase {
		_ = syscall.Kill(os.Getpid(), syscall.SIGKILL)
	}
}

func TestVMKilledProcess(t *testing.T) {
	q, err := Inspect()
	if err != nil || q.Status != "SUPPORTED WITH WARNINGS" || os.Geteuid() != 0 {
		t.Fatal("disposable VM required", q, err)
	}
	if os.Getenv("GRUBMGR_VM_KILL_PHASE") != "" {
		pl, err := build("/", planner.Request{Action: "switch", Target: "grubmgr/cyberpunk-demo", Variant: "hd"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = apply(Request{Operation: "apply", Plan: pl.ID}, transaction.Faults{})
		t.Fatal("process was not killed", err)
	}
	beforeD, _ := os.ReadFile("/" + Defaults)
	beforeC, _ := os.ReadFile("/" + Config)
	for _, phase := range []string{"settings_staged", "activated"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestVMKilledProcess$")
		cmd.Env = append(os.Environ(), "GRUBMGR_VM_KILL_PHASE="+phase)
		err := cmd.Run()
		ee, ok := err.(*exec.ExitError)
		if !ok || ee.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
			t.Fatal("expected SIGKILL", err)
		}
		if _, err = recoverAll(); err != nil {
			t.Fatal(err)
		}
		d, _ := os.ReadFile("/" + Defaults)
		c, _ := os.ReadFile("/" + Config)
		if !bytes.Equal(d, beforeD) || !bytes.Equal(c, beforeC) {
			t.Fatal("SIGKILL recovery mismatch")
		}
	}
}

func TestVMDiskFullReplacement(t *testing.T) {
	q, err := Inspect()
	if err != nil || q.Status != "SUPPORTED WITH WARNINGS" || os.Geteuid() != 0 {
		t.Fatal("disposable VM required", q, err)
	}
	dir, err := os.MkdirTemp("/var/lib/grubmgr", "diskfull-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if out, err := exec.Command("/usr/bin/mount", "-t", "tmpfs", "-o", "size=1m", "tmpfs", dir).CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	defer func() {
		if out, err := exec.Command("/usr/bin/umount", dir).CombinedOutput(); err != nil {
			t.Error(err, string(out))
		}
	}()
	name := filepath.Join(dir, "known-good")
	if err = os.WriteFile(name, []byte("known good"), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := os.OpenRoot("/")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err = atomicReplace(r, name[1:], make([]byte, 2<<20)); err == nil {
		t.Fatal("disk-full write succeeded")
	}
	b, err := os.ReadFile(name)
	if err != nil || string(b) != "known good" {
		t.Fatal("disk full damaged existing file", err)
	}
}

func TestVMRollbackFailure(t *testing.T) {
	q, err := Inspect()
	if err != nil || q.Status != "SUPPORTED WITH WARNINGS" || os.Geteuid() != 0 {
		t.Fatal("disposable VM required", q, err)
	}
	pl, err := build("/", planner.Request{Action: "switch", Target: "grubmgr/cyberpunk-demo", Variant: "hd"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	switched, err := apply(Request{Operation: "apply", Plan: pl.ID}, transaction.Faults{})
	if err != nil {
		t.Fatal(err)
	}
	d, _ := os.ReadFile("/" + Defaults)
	c, _ := os.ReadFile("/" + Config)
	pl, err = build("/", planner.Request{Action: "rollback", Target: switched.ID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := apply(Request{Operation: "apply", Plan: pl.ID}, transaction.Faults{FailAfter: "candidate_validated"})
	if err == nil || tx.Phase != "rolled_back" {
		t.Fatal("failed rollback not restored", err)
	}
	afterD, _ := os.ReadFile("/" + Defaults)
	afterC, _ := os.ReadFile("/" + Config)
	if !bytes.Equal(d, afterD) || !bytes.Equal(c, afterC) {
		t.Fatal("failed rollback changed boot files")
	}
	pl, err = build("/", planner.Request{Action: "rollback", Target: switched.ID}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = apply(Request{Operation: "apply", Plan: pl.ID}, transaction.Faults{}); err != nil {
		t.Fatal(err)
	}
}

func TestVMFailures(t *testing.T) {
	q, err := Inspect()
	if err != nil || q.Status != "SUPPORTED WITH WARNINGS" || os.Geteuid() != 0 {
		t.Fatal("disposable VM required", q, err)
	}
	r, err := os.OpenRoot("/")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	read := func(name string) []byte {
		b, err := fsx.Read(r, name, 8<<20)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	t.Run("generator_failure", func(t *testing.T) {
		name := "/etc/grub.d/09_grubmgr_test_failure"
		if err := os.WriteFile(name, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(name)
		pl, err := build("/", planner.Request{Action: "switch", Target: "grubmgr/cyberpunk-demo", Variant: "hd"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		d, c := read(Defaults), read(Config)
		tx, err := apply(Request{Operation: "apply", Plan: pl.ID}, transaction.Faults{})
		if err == nil || tx.Phase != "rolled_back" {
			t.Fatal(tx.Phase, err)
		}
		if !bytes.Equal(d, read(Defaults)) || !bytes.Equal(c, read(Config)) {
			t.Fatal("generator failure did not restore")
		}
	})
	for _, phase := range transaction.Phases {
		t.Run("failure_"+phase, func(t *testing.T) {
			pl, err := build("/", planner.Request{Action: "switch", Target: "grubmgr/cyberpunk-demo", Variant: "hd"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defaults, cfg := read(Defaults), read(Config)
			tx, err := apply(Request{Operation: "apply", Plan: pl.ID}, transaction.Faults{FailAfter: phase})
			if err == nil || tx.Phase != "rolled_back" {
				t.Fatal(tx.Phase, err)
			}
			if !bytes.Equal(defaults, read(Defaults)) || !bytes.Equal(cfg, read(Config)) {
				t.Fatal("failed to restore exact boot state")
			}
		})
	}
	for _, phase := range transaction.Phases[:len(transaction.Phases)-1] {
		t.Run("interruption_"+phase, func(t *testing.T) {
			pl, err := build("/", planner.Request{Action: "switch", Target: "grubmgr/cyberpunk-demo", Variant: "hd"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defaults, cfg := read(Defaults), read(Config)
			tx, err := apply(Request{Operation: "apply", Plan: pl.ID}, transaction.Faults{CrashAfter: phase})
			if err == nil || tx.Phase != phase {
				t.Fatal(tx.Phase, err)
			}
			if _, err = build("/", pl.Request, nil); err == nil {
				t.Fatal("pending journal did not block planning")
			}
			if _, err = recoverAll(); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(defaults, read(Defaults)) || !bytes.Equal(cfg, read(Config)) {
				t.Fatal("interrupted transaction recovery differs")
			}
		})
	}
	t.Run("stale_plan", func(t *testing.T) {
		pl, err := build("/", planner.Request{Action: "switch", Target: "grubmgr/cyberpunk-demo", Variant: "hd"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		before := read(Defaults)
		defer atomicReplace(r, Defaults, before)
		if err = atomicReplace(r, Defaults, append(append([]byte{}, before...), []byte("# manual VM test edit\n")...)); err != nil {
			t.Fatal(err)
		}
		if _, err = apply(Request{Operation: "apply", Plan: pl.ID}, transaction.Faults{}); err == nil {
			t.Fatal("stale plan accepted")
		}
	})
	t.Run("destination_ownership", func(t *testing.T) {
		dir := "/boot/grub/themes/grubmgr"
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.Chmod(dir, 0777); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(dir, info.Mode().Perm())
		if _, err = build("/", planner.Request{Action: "switch", Target: "grubmgr/cyberpunk-demo", Variant: "hd"}, nil); err == nil {
			t.Fatal("writable destination ancestor accepted")
		}
	})
	t.Run("asset_permissions", func(t *testing.T) {
		pl, err := build("/", planner.Request{Action: "switch", Target: "grubmgr/cyberpunk-demo", Variant: "hd"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(pl.Destination, "theme.txt")
		info, err := os.Stat(name)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.Chmod(name, 0666); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(name, info.Mode().Perm())
		if _, err = build("/", pl.Request, nil); err == nil {
			t.Fatal("writable theme file accepted")
		}
	})
	t.Run("recovery_conflict", func(t *testing.T) {
		pl, err := build("/", planner.Request{Action: "switch", Target: "grubmgr/cyberpunk-demo", Variant: "hd"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		tx, err := apply(Request{Operation: "apply", Plan: pl.ID}, transaction.Faults{CrashAfter: "activated"})
		if err == nil {
			t.Fatal("expected interruption")
		}
		written := read(Defaults)
		if err = atomicReplace(r, Defaults, append(append([]byte{}, written...), []byte("# conflicting administrator edit\n")...)); err != nil {
			t.Fatal(err)
		}
		if _, err = recoverAll(); err == nil {
			t.Fatal("conflicting edit silently restored")
		}
		db, err := state.Open(paths("/"), false)
		if err != nil {
			t.Fatal(err)
		}
		history, err := db.History()
		db.Close()
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, item := range history {
			if item.ID == tx.ID && item.Phase == "recovery_required" && model.Hash(item.Before.Defaults) == model.Hash(pl.Before.Defaults) {
				found = true
			}
		}
		if !found {
			t.Fatal("missing retained recovery state")
		}
		if err = atomicReplace(r, Defaults, written); err != nil {
			t.Fatal(err)
		}
		if _, err = recoverAll(); err != nil {
			t.Fatal(err)
		}
	})
}
