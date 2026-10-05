package state

import (
	"grubmgr/internal/testutil"
	"os"
	"path/filepath"
	"testing"
)

func TestReadOnlyAndLock(t *testing.T) {
	p := testutil.Root(t, "debian")
	s, e := Open(p, false)
	if e != nil {
		t.Fatal(e)
	}
	if list, e := s.Packages(); e != nil || len(list) != 0 {
		t.Fatal(list, e)
	}
	s.Close()
	if _, e = os.Stat(filepath.Join(p.Root, ".grubmgr")); !os.IsNotExist(e) {
		t.Fatal("read-only store created files")
	}
	unlock, e := Lock(p)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = Lock(p); e == nil {
		t.Fatal("concurrent state lock accepted")
	}
	unlock()
	unlock, e = Lock(p)
	if e != nil {
		t.Fatal(e)
	}
	unlock()
}
func TestNewerSchemaRefused(t *testing.T) {
	p := testutil.Root(t, "debian")
	s, e := Open(p, true)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("PRAGMA user_version=2"); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if s, e = Open(p, false); e == nil {
		s.Close()
		t.Fatal("future schema accepted")
	}
}
