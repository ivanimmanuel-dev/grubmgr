package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestZIPPolicy(t *testing.T) {
	cases := []struct {
		name   string
		names  []string
		mode   os.FileMode
		limits Limits
		ok     bool
	}{
		{"valid", []string{"nested/theme.txt", "nested/assets/a.png"}, 0600, Default, true},
		{"traversal", []string{"../outside"}, 0600, Default, false},
		{"absolute", []string{"/outside"}, 0600, Default, false},
		{"windows", []string{"C:/outside"}, 0600, Default, false},
		{"backslash", []string{`a\b`}, 0600, Default, false},
		{"duplicate", []string{"theme.txt", "theme.txt"}, 0600, Default, false},
		{"case", []string{"A/theme.txt", "a/font.pf2"}, 0600, Default, false},
		{"link", []string{"link"}, os.ModeSymlink | 0777, Default, false},
		{"device", []string{"special"}, os.ModeDevice | 0600, Default, false},
		{"oversized", []string{"theme.txt"}, 0600, Limits{10, 2, 100}, false},
		{"total", []string{"a", "b"}, 0600, Limits{10, 100, 6}, false},
		{"count", []string{"a", "b"}, 0600, Limits{1, 100, 100}, false},
		{"reserved", []string{"nul.txt"}, 0600, Default, false},
		{"conflict", []string{"a", "a/b"}, 0600, Default, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			z := zip.NewWriter(&buf)
			for _, n := range tt.names {
				h := &zip.FileHeader{Name: n, Method: zip.Deflate}
				h.SetMode(tt.mode)
				w, e := z.CreateHeader(h)
				if e != nil {
					t.Fatal(e)
				}
				_, _ = w.Write([]byte("hello"))
			}
			if e := z.Close(); e != nil {
				t.Fatal(e)
			}
			file := filepath.Join(t.TempDir(), "test.zip")
			os.WriteFile(file, buf.Bytes(), 0600)
			dest := t.TempDir()
			err := Extract(file, dest, tt.limits)
			if (err == nil) != tt.ok {
				t.Fatalf("ok=%v, error=%v", tt.ok, err)
			}
		})
	}
}
func TestTARPolicy(t *testing.T) {
	for _, kind := range []byte{tar.TypeReg, tar.TypeSymlink, tar.TypeLink, tar.TypeChar, tar.TypeFifo} {
		t.Run(string(kind), func(t *testing.T) {
			var b bytes.Buffer
			w := tar.NewWriter(&b)
			h := &tar.Header{Name: "theme.txt", Typeflag: kind, Mode: 0600, Linkname: "../outside"}
			if kind == tar.TypeReg {
				h.Size = 3
			}
			if e := w.WriteHeader(h); e != nil {
				t.Fatal(e)
			}
			if kind == tar.TypeReg {
				w.Write([]byte("abc"))
			}
			w.Close()
			file := filepath.Join(t.TempDir(), "theme.tar")
			os.WriteFile(file, b.Bytes(), 0600)
			e := Extract(file, t.TempDir(), Default)
			if (e == nil) != (kind == tar.TypeReg) {
				t.Fatal(e)
			}
		})
	}
}
