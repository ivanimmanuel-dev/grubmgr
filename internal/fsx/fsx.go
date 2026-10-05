// Package fsx provides bounded, portable, link-free package I/O.
package fsx

import (
	"fmt"
	"grubmgr/internal/model"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const MaxFile int64 = 16 << 20
const MaxTree int64 = 128 << 20
const MaxMembers = 4096

func SafePath(p string) error {
	if p == "" || p == "." || strings.ContainsAny(p, "\\:\x00") || strings.HasPrefix(p, "/") || path.Clean(p) != p || len(p) > 240 || strings.Count(p, "/") > 32 {
		return fmt.Errorf("unsafe or noncanonical path %q", p)
	}
	for _, s := range strings.Split(p, "/") {
		if s == ".." || s == "." || strings.TrimRight(s, " .") != s {
			return fmt.Errorf("unsafe component %q", s)
		}
		for _, c := range s {
			if c < 32 || c > 126 || strings.ContainsRune(`<>"|?*`, c) {
				return fmt.Errorf("nonportable component %q", s)
			}
		}
		base := strings.ToUpper(strings.Split(s, ".")[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9' {
			return fmt.Errorf("reserved component %q", s)
		}
	}
	return nil
}
func NoLinks(abs string) error {
	abs, err := filepath.Abs(abs)
	if err != nil {
		return err
	}
	for p := abs; ; p = filepath.Dir(p) {
		i, e := os.Lstat(p)
		if e == nil && i.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("links are forbidden: %s", p)
		}
		if e == nil && i.Mode().IsRegular() {
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			err = SingleLink(f)
			f.Close()
			if err != nil {
				return fmt.Errorf("%s: %w", p, err)
			}
		}
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return nil
}
func Read(r *os.Root, p string, limit int64) ([]byte, error) {
	f, e := r.Open(p)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	i, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if !i.Mode().IsRegular() || i.Size() > limit {
		return nil, fmt.Errorf("not a bounded regular file: %s", p)
	}
	b, e := io.ReadAll(io.LimitReader(f, limit+1))
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("file exceeds limit")
	}
	return b, e
}
func Write(r *os.Root, p string, b []byte) error {
	if e := SafePath(p); e != nil {
		return e
	}
	if e := r.MkdirAll(path.Dir(p), 0700); e != nil {
		return e
	}
	f, e := r.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return e
	}
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	return ce
}

// Replace is only used inside explicit fixture roots. Rename is on the same filesystem.
func Replace(r *os.Root, p string, b []byte) error {
	// Windows os.Root.Rename replacement is not available in some restricted
	// environments. Fixture-only writes use the already durable recovery journal.
	// This fallback is intentionally not a future real activation primitive.
	if runtime.GOOS == "windows" {
		if e := SafePath(p); e != nil {
			return e
		}
		f, e := r.OpenFile(p, os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		if e = SingleLink(f); e != nil {
			f.Close()
			return e
		}
		if e = f.Truncate(0); e != nil {
			f.Close()
			return e
		}
		_, e = f.Write(b)
		if e == nil {
			e = f.Sync()
		}
		ce := f.Close()
		if e != nil {
			return e
		}
		return ce
	}
	tmp := p + ".grubmgr-new"
	if e := Write(r, tmp, b); e != nil {
		return e
	}
	if e := r.Rename(tmp, p); e != nil {
		_ = r.Remove(tmp)
		return e
	}
	return nil
}
func Inventory(dir string) ([]model.File, string, error) {
	if e := NoLinks(dir); e != nil {
		return nil, "", e
	}
	r, e := os.OpenRoot(dir)
	if e != nil {
		return nil, "", e
	}
	defer r.Close()
	files := []model.File{}
	seen := map[string]string{}
	var total int64
	members := 0
	e = fs.WalkDir(r.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == "." {
			return nil
		}
		members++
		if members > MaxMembers {
			return fmt.Errorf("member limit exceeded")
		}
		if err = SafePath(p); err != nil {
			return err
		}
		key := strings.ToLower(p)
		if old, ok := seen[key]; ok {
			return fmt.Errorf("duplicate/case-conflicting paths: %s, %s", old, p)
		}
		seen[key] = p
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("link rejected: %s", p)
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("special file rejected: %s", p)
		}
		b, err := Read(r, p, MaxFile)
		if err != nil {
			return err
		}
		total += int64(len(b))
		if total > MaxTree {
			return fmt.Errorf("tree byte limit exceeded")
		}
		files = append(files, model.File{Path: p, Size: int64(len(b)), SHA256: model.Hash(b)})
		return nil
	})
	if e != nil {
		return nil, "", e
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, model.Digest(files), nil
}
func CopyTree(src string, dst *os.Root, prefix string) error {
	files, _, e := Inventory(src)
	if e != nil {
		return e
	}
	s, e := os.OpenRoot(src)
	if e != nil {
		return e
	}
	defer s.Close()
	for _, f := range files {
		b, e := Read(s, f.Path, MaxFile)
		if e != nil {
			return e
		}
		if e = Write(dst, path.Join(prefix, f.Path), b); e != nil {
			return e
		}
	}
	return nil
}
