package debian

import (
	"grubmgr/internal/model"
	"testing"
)

func TestStatusReflectsActualThemeSelection(t *testing.T) {
	packages := []model.Package{{Manifest: model.Manifest{Recipe: model.Recipe{ID: "author/theme", Root: ".", Entry: "theme.txt"}, Revision: "revision"}, Installed: true, Active: true}}
	got, err := activePackages(packages, "/boot/other/theme.txt")
	if err != nil || got[0].Active {
		t.Fatal("stale receipt reported as active", got, err)
	}
	got, err = activePackages(packages, "/boot/grub/themes/grubmgr/author/theme/revision/theme.txt")
	if err != nil || !got[0].Active {
		t.Fatal("actual theme not detected", got, err)
	}
}
