package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"grubmgr/internal/fsx"
	"io"
	"os"
	"path"
	"strings"
)

type Limits struct {
	Members               int
	FileBytes, TotalBytes int64
}

var Default = Limits{fsx.MaxMembers, fsx.MaxFile, fsx.MaxTree}

type extractor struct {
	root     *os.Root
	limits   Limits
	seen     map[string]string
	explicit map[string]bool
	count    int
	total    int64
}

func (x *extractor) member(name string, dir bool, size int64, reader io.Reader) error {
	if dir {
		name = strings.TrimSuffix(name, "/")
	}
	if e := fsx.SafePath(name); e != nil {
		return e
	}
	x.count++
	if x.count > x.limits.Members || size < 0 || size > x.limits.FileBytes || x.total+size > x.limits.TotalBytes {
		return fmt.Errorf("archive limit exceeded")
	}
	key := strings.ToLower(name)
	if x.explicit[key] {
		return fmt.Errorf("duplicate archive member %s", name)
	}
	x.explicit[key] = true
	for p := name; p != "."; p = path.Dir(p) {
		k := strings.ToLower(p)
		if old, ok := x.seen[k]; ok && old != p {
			return fmt.Errorf("case-conflicting archive path %s", name)
		}
		x.seen[k] = p
	}
	if dir {
		return x.root.MkdirAll(name, 0700)
	}
	b, e := io.ReadAll(io.LimitReader(reader, min(x.limits.FileBytes, x.limits.TotalBytes-x.total)+1))
	if e != nil {
		return e
	}
	if int64(len(b)) != size || int64(len(b)) > x.limits.FileBytes {
		return fmt.Errorf("archive member size mismatch/limit: %s", name)
	}
	x.total += int64(len(b))
	return fsx.Write(x.root, name, b)
}

// Extract never dispatches external tools. All links, even internal links, are rejected.
func Extract(file, dest string, limits Limits) error {
	r, e := os.OpenRoot(dest)
	if e != nil {
		return e
	}
	defer r.Close()
	x := extractor{root: r, limits: limits, seen: map[string]string{}, explicit: map[string]bool{}}
	f, e := os.Open(file)
	if e != nil {
		return e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return e
	}
	if info.Size() > fsx.MaxTree {
		return fmt.Errorf("archive compressed-size limit exceeded")
	}
	lower := strings.ToLower(file)
	if strings.HasSuffix(lower, ".zip") {
		z, e := zip.NewReader(f, info.Size())
		if e != nil {
			return e
		}
		for _, m := range z.File {
			if m.Mode()&os.ModeSymlink != 0 || !m.Mode().IsRegular() && !m.FileInfo().IsDir() {
				return fmt.Errorf("special ZIP member %s", m.Name)
			}
			if m.UncompressedSize64 > uint64(limits.FileBytes) {
				return fmt.Errorf("archive file-size limit")
			}
			rc, e := m.Open()
			if e != nil {
				return e
			}
			e = x.member(m.Name, m.FileInfo().IsDir(), int64(m.UncompressedSize64), rc)
			rc.Close()
			if e != nil {
				return e
			}
		}
		return nil
	}
	var reader io.Reader = f
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		g, e := gzip.NewReader(f)
		if e != nil {
			return e
		}
		defer g.Close()
		reader = g
	} else if !strings.HasSuffix(lower, ".tar") {
		return fmt.Errorf("supported archives: ZIP, TAR, TAR.GZ")
	}
	t := tar.NewReader(reader)
	for {
		h, e := t.Next()
		if e == io.EOF {
			return nil
		}
		if e != nil {
			return e
		}
		if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeDir {
			return fmt.Errorf("special TAR member %s", h.Name)
		}
		if len(h.PAXRecords) > 0 {
			return fmt.Errorf("extended TAR metadata is not supported")
		}
		if e = x.member(h.Name, h.Typeflag == tar.TypeDir, h.Size, t); e != nil {
			return e
		}
	}
}
