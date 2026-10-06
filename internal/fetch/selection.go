package fetch

import (
	"fmt"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"os"
	"strings"
)

// Selection happens only in the unprivileged extraction directory. Declared
// notices are retained even when they live outside the selected theme folders.
func selectFiles(directory string, recipe model.Recipe) error {
	if len(recipe.Include) == 0 {
		return nil
	}
	selected := append(append([]string{}, recipe.Include...), recipe.License.Notices...)
	r, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, name := range selected {
		if err := fsx.SafePath(name); err != nil {
			return fmt.Errorf("invalid include path %q: %w", name, err)
		}
		if _, err := r.Stat(name); err != nil {
			return fmt.Errorf("included path %q: %w", name, err)
		}
	}
	files, _, err := fsx.Inventory(directory)
	if err != nil {
		return err
	}
	for _, file := range files {
		keep := false
		for _, name := range selected {
			if file.Path == name || strings.HasPrefix(file.Path, name+"/") {
				keep = true
				break
			}
		}
		if !keep {
			if err := r.Remove(file.Path); err != nil {
				return err
			}
		}
	}
	return nil
}
