package preview

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestScreenshotConversion(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "frame.ppm"), filepath.Join(dir, "frame.png")
	if err := os.WriteFile(src, append([]byte("P6\n1 1\n255\n"), 255, 0, 128), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ppmToPNG(src, dst); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, a := img.At(0, 0).RGBA()
	if r != 65535 || g != 0 || b != 32896 || a != 65535 {
		t.Fatal(r, g, b, a)
	}
}

func TestScreenshotBounds(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "frame.ppm")
	for _, body := range []string{"P6\n999999 1\n255\n", "P6\n1 1\n255\n", "P3\n1 1\n255\n"} {
		if err := os.WriteFile(src, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if ppmToPNG(src, filepath.Join(dir, "out.png")) == nil {
			t.Fatal("accepted invalid screenshot")
		}
	}
	blank := append([]byte("P6\n640 480\n255\n"), make([]byte, 640*480*3)...)
	if err := os.WriteFile(src, blank, 0600); err != nil {
		t.Fatal(err)
	}
	if ppmToPNG(src, filepath.Join(dir, "blank.png")) == nil {
		t.Fatal("blank frame reported as a preview")
	}
}
