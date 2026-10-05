package fetch

import (
	"fmt"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/system"
	"os"
	"path/filepath"
	"strings"
)

// importDebian copies one pinned, installed Debian data package. It runs no
// package manager or maintainer script and resolves all sources beneath p.Root.
func importDebian(p system.Paths, dst *os.Root, recipe model.Recipe) error {
	if recipe.ID != "debian/starfield" || recipe.Source.URL != "debian:grub-theme-starfield=2.12-9+deb13u2" {
		return fmt.Errorf("unsupported Debian data package")
	}
	r, err := os.OpenRoot(p.Root)
	if err != nil {
		return err
	}
	defer r.Close()
	status, err := fsx.Read(r, "var/lib/dpkg/status", 32<<20)
	if err != nil {
		return err
	}
	found := false
	for _, stanza := range strings.Split(string(status), "\n\n") {
		if strings.HasPrefix(stanza, "Package: grub-theme-starfield\n") && strings.Contains(stanza, "\nStatus: install ok installed\n") && strings.Contains(stanza, "\nVersion: 2.12-9+deb13u2\n") {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("requires the installed Debian package grub-theme-starfield 2.12-9+deb13u2")
	}
	if err = fsx.CopyTree(filepath.Join(p.Root, "usr/share/grub/themes/starfield"), dst, ""); err != nil {
		return err
	}
	for name, source := range map[string]string{"DEBIAN-COPYRIGHT": "usr/share/doc/grub-theme-starfield/copyright", "FONT-COPYRIGHT": "usr/share/doc/fonts-dejavu-core/copyright", "GPL-3": "usr/share/common-licenses/GPL-3"} {
		b, err := fsx.Read(r, source, 1<<20)
		if err != nil {
			return err
		}
		if err = fsx.Write(dst, name, b); err != nil {
			return err
		}
	}
	return nil
}
