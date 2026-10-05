package testutil

import (
	"grubmgr/internal/fsx"
	"grubmgr/internal/system"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func Root(t testing.TB, name string) system.Paths {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	source := filepath.Join(filepath.Dir(file), "../../fixtures", name)
	dir := t.TempDir()
	r, e := os.OpenRoot(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	if e = fsx.CopyTree(source, r, ""); e != nil {
		t.Fatal(e)
	}
	p, e := system.Locations(dir)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func Write(t testing.TB, dir, name string, b []byte) {
	t.Helper()
	f := filepath.Join(dir, filepath.FromSlash(name))
	if e := os.MkdirAll(filepath.Dir(f), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(f, b, 0600); e != nil {
		t.Fatal(e)
	}
}
func Read(t testing.TB, dir, name string) []byte {
	t.Helper()
	b, e := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if e != nil {
		t.Fatal(e)
	}
	return b
}
