package fetch

import (
	"grubmgr/internal/catalog"
	"grubmgr/internal/system"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDebianImportNeverFallsBackToHost(t *testing.T) {
	entry, err := catalog.Find("debian/starfield")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	dst, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	p := system.Paths{Root: root}
	if err = importDebian(p, dst, entry.Recipe); err == nil {
		t.Fatal("missing fixture package accepted")
	}
	if err = os.MkdirAll(filepath.Join(root, "var/lib/dpkg"), 0700); err != nil {
		t.Fatal(err)
	}
	status := "Package: grub-theme-starfield\nStatus: install ok installed\nVersion: wrong-version\n\n"
	if err = os.WriteFile(filepath.Join(root, "var/lib/dpkg/status"), []byte(status), 0600); err != nil {
		t.Fatal(err)
	}
	if err = importDebian(p, dst, entry.Recipe); err == nil || !strings.Contains(err.Error(), "2.12-9+deb13u2") {
		t.Fatal("wrong version accepted", err)
	}
	entry.Recipe.Source.URL = "debian:caller-chosen-package"
	if err = importDebian(p, dst, entry.Recipe); err == nil {
		t.Fatal("unapproved installed-package source accepted")
	}
}

func TestTsushimaRemainsBrowseOnly(t *testing.T) {
	entry, err := catalog.Find("grub-of-tsushima")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Recipe.Reviewed || entry.Recipe.License.Verified {
		t.Fatal("unconfirmed asset rights approved")
	}
	if _, err = Import(system.Paths{Root: t.TempDir()}, "grub-of-tsushima", ""); err == nil || !strings.Contains(err.Error(), "no reviewed package recipe") {
		t.Fatal("browse-only theme was fetched", err)
	}
}
