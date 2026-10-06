package debian

import (
	"grubmgr/internal/model"
	"path"
	"strings"
	"testing"
)

func TestOriginalBackupsCannotOverlapPackageDestinations(t *testing.T) {
	digest := strings.Repeat("a", 64)
	backup := originalDestination(model.Manifest{TreeSHA256: digest})
	id := strings.TrimPrefix(backup, "/boot/grub/themes/grubmgr/")
	recipe := model.Recipe{Schema: 1, ID: id, Name: "Collision", RecipeRevision: "1"}
	if recipe.Check() == nil {
		t.Fatal("a package ID can contain a backup directory", id)
	}
	if path.Dir(backup) != "/boot/grub/themes/grubmgr/.backups" {
		t.Fatal("backup destination escaped its reserved directory", backup)
	}
}
