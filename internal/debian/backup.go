package debian

import (
	"fmt"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/validate"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func originalDestination(m model.Manifest) string {
	return "/boot/grub/themes/grubmgr/.backups/" + m.TreeSHA256
}

func originalSnapshot(root, theme string) (*model.Manifest, error) {
	if !strings.HasPrefix(theme, "/") || fsx.SafePath(strings.TrimPrefix(theme, "/")) != nil {
		return nil, fmt.Errorf("previous theme needs a canonical absolute entry path")
	}
	directory, err := os.MkdirTemp(paths(root).Cache, ".original-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	if err := copyOriginal(root, theme, directory); err != nil {
		return nil, err
	}
	files, digest, err := fsx.Inventory(directory)
	if err != nil {
		return nil, err
	}
	m := model.Manifest{Recipe: model.Recipe{
		Schema: 1, ID: "system/previous-theme", Name: "Previous theme", Version: "snapshot",
		Source: model.Source{Provider: "system", URL: theme, UpstreamRevision: digest},
		Root:   ".", Entry: path.Base(theme), RecipeRevision: "1", ArtifactSHA256: digest,
	}, Files: files, TreeSHA256: digest}
	m.Revision = m.Identity()
	if report := validate.Run(directory, m); !report.Valid {
		return nil, fmt.Errorf("cannot back up the previous theme: %v", report.Findings)
	}
	return &m, nil
}

func verifyOriginal(m model.Manifest) error {
	if !model.IsDigest(m.TreeSHA256) || m.Identity() != m.Revision || fsx.SafePath(m.Entry) != nil {
		return fmt.Errorf("invalid previous theme snapshot")
	}
	directory := originalDestination(m)
	if err := protectedDestination(directory); err != nil {
		return err
	}
	if report := validate.Run(directory, m); !report.Valid {
		return fmt.Errorf("previous theme backup failed verification: %v", report.Findings)
	}
	return nil
}

func backupOriginal(root string, m *model.Manifest) error {
	if m == nil {
		return nil
	}
	destination := originalDestination(*m)
	if _, err := os.Stat(destination); err == nil {
		return verifyOriginal(*m)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := protectedDestination(destination); err != nil {
		return err
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".backup-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if err = copyOriginal(root, m.Source.URL, staging); err != nil {
		return err
	}
	if report := validate.Run(staging, *m); !report.Valid {
		return fmt.Errorf("previous theme changed before backup")
	}
	if err = syncTree(staging); err != nil {
		return err
	}
	if err = os.Rename(staging, destination); err != nil {
		return err
	}
	f, err := os.Open(parent)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// copyOriginal copies root-owned font links into regular backup files.
func copyOriginal(root, theme, destination string) error {
	directory, err := filepath.EvalSymlinks(path.Join(root, path.Dir(theme)))
	if err != nil {
		return err
	}
	if err := secure(directory, true); err != nil {
		return err
	}
	dest, err := os.OpenRoot(destination)
	if err != nil {
		return err
	}
	defer dest.Close()
	var total int64
	count := 0
	return filepath.WalkDir(directory, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return secure(name, true)
		}
		resolved, err := filepath.EvalSymlinks(name)
		if err != nil {
			return err
		}
		if err := secure(resolved, false); err != nil {
			return err
		}
		src, err := os.OpenRoot(filepath.Dir(resolved))
		if err != nil {
			return err
		}
		data, err := fsx.Read(src, filepath.Base(resolved), fsx.MaxFile)
		src.Close()
		if err != nil {
			return err
		}
		count++
		total += int64(len(data))
		if count > fsx.MaxMembers || total > fsx.MaxTree {
			return fmt.Errorf("previous theme exceeds backup limits")
		}
		relative, err := filepath.Rel(directory, name)
		if err != nil {
			return err
		}
		return fsx.Write(dest, filepath.ToSlash(relative), data)
	})
}
