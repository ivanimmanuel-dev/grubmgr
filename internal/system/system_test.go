package system_test

import (
	"grubmgr/internal/fsx"
	"grubmgr/internal/system"
	"grubmgr/internal/testutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDetection(t *testing.T) {
	for _, tt := range []struct{ name, status, backend string }{{"debian", "fixture-only", "debian-conventional"}, {"arch", "read-only", "arch-conventional"}, {"fedora", "read-only", "fedora-conventional"}, {"non-grub", "unsupported", "none"}, {"ambiguous", "ambiguous", "none"}} {
		t.Run(tt.name, func(t *testing.T) {
			p := testutil.Root(t, tt.name)
			_, before, _ := fsx.Inventory(p.Root)
			q, e := system.Inspect(p)
			if e != nil || q.Status != tt.status || q.Backend != tt.backend {
				t.Fatalf("%+v %v", q, e)
			}
			_, after, _ := fsx.Inventory(p.Root)
			if before != after {
				t.Fatal("doctor wrote files")
			}
			if q.Architecture != "x86_64" {
				t.Fatal("fixture architecture fell back to host")
			}
		})
	}
}
func TestNoHostFallback(t *testing.T) {
	p, _ := system.Locations(t.TempDir())
	q, e := system.Inspect(p)
	if e != nil || q.Distribution != "unknown" || q.Architecture != "unknown" || len(q.Utilities) != 0 {
		t.Fatal(q, e)
	}
}
func TestRefuseSymlinkRoot(t *testing.T) {
	outside := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if e := os.Symlink(outside, link); e != nil {
		t.Skip("symlink creation unavailable:", e)
	}
	if _, e := system.Locations(link); e == nil {
		t.Fatal("symlink root accepted")
	}
}
func TestThemeParser(t *testing.T) {
	for _, s := range []string{"GRUB_THEME=$(x)", "export GRUB_THEME=/a", "GRUB_THEME=/a\nGRUB_THEME=/b", "GRUB_THEME=\"$HOME/theme\""} {
		if _, ok := system.Theme([]byte(s)); ok {
			t.Fatal("complex shell accepted", s)
		}
	}
}
