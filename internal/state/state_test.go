package state

import (
	"grubmgr/internal/model"
	"grubmgr/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRevisionSelectorsRejectAmbiguity(t *testing.T) {
	first := model.Package{Manifest: model.Manifest{Recipe: model.Recipe{ID: "author/theme"}, Revision: strings.Repeat("a", 64)}}
	second := first
	second.Manifest.Revision = strings.Repeat("a", 12) + strings.Repeat("b", 52)
	packages := []model.Package{first, second}
	if _, err := Select(packages, "author/theme@"+strings.Repeat("a", 12)); err == nil {
		t.Fatal("ambiguous revision prefix selected")
	}
	got, err := Select(packages, "author/theme@"+strings.Repeat("a", 13))
	if err != nil || got.Manifest.Revision != first.Manifest.Revision {
		t.Fatal(got, err)
	}
	for _, selector := range []string{"author/theme@aaaa", "author/theme@../../etc", "other/theme@" + strings.Repeat("a", 13)} {
		if _, err := Select(packages, selector); err == nil {
			t.Fatal("invalid selector accepted", selector)
		}
	}
}

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
