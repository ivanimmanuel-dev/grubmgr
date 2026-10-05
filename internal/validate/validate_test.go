package validate

import (
	"bytes"
	"encoding/binary"
	"grubmgr/internal/model"
	"grubmgr/internal/testutil"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"testing"
)

func recipe() model.Manifest {
	return model.Manifest{Recipe: model.Recipe{Schema: 1, ID: "test/theme", Name: "Test", Root: ".", Entry: "theme.txt", RecipeRevision: "1"}}
}
func TestThemes(t *testing.T) {
	var img bytes.Buffer
	palette := image.NewPaletted(image.Rect(0, 0, 2, 2), color.Palette{color.Black, color.White})
	png.Encode(&img, palette)
	cases := []struct {
		name, theme string
		files       map[string][]byte
		ok          bool
		code        string
	}{
		{"valid", "title-text: \"Synthetic\"\n", nil, true, "UNKNOWN_LICENSE"},
		{"nested", "desktop-image: \"assets/nested/bg.png\"\n", map[string][]byte{"assets/nested/bg.png": img.Bytes()}, true, ""},
		{"missing", "desktop-image: \"missing.png\"\n", nil, false, "ASSET_MISSING"},
		{"escape", "desktop-image: \"../bg.png\"\n", nil, false, "ASSET_ESCAPE"},
		{"case", "desktop-image: \"bg.png\"\n", map[string][]byte{"BG.png": img.Bytes()}, false, "ASSET_MISSING"},
		{"duplicate", "title-text: \"x\"\n", map[string][]byte{"A.txt": {}, "a.txt": {}}, false, "PACKAGE_PATH"},
		{"bad-image", "desktop-image: \"bg.png\"\n", map[string][]byte{"bg.png": []byte("bad")}, false, "IMAGE_DECODE"},
		{"script", "title-text: \"x\"\n", map[string][]byte{"install.sh": []byte("not executed")}, false, "EXECUTABLE_DATA"},
		{"font-missing", "title-font: \"Missing Regular 16\"\n", nil, false, "FONT_MISSING"},
		{"braces", "+ boot_menu {\n", nil, false, "THEME_SYNTAX"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "duplicate" && runtime.GOOS == "windows" {
				t.Skip("case-insensitive filesystem; ZIP path collision test covers Windows")
			}
			d := t.TempDir()
			testutil.Write(t, d, "theme.txt", []byte(tt.theme))
			for p, b := range tt.files {
				testutil.Write(t, d, p, b)
			}
			v := Run(d, recipe())
			if v.Valid != tt.ok {
				t.Fatalf("%+v", v)
			}
			if tt.code != "" {
				found := false
				for _, f := range v.Findings {
					found = found || f.Code == tt.code
				}
				if !found {
					t.Fatalf("missing code %s: %+v", tt.code, v)
				}
			}
		})
	}
}
func TestPF2AndVariants(t *testing.T) {
	d := t.TempDir()
	section := func(tag string, b []byte) []byte {
		x := append([]byte(tag), 0, 0, 0, 0)
		binary.BigEndian.PutUint32(x[4:], uint32(len(b)))
		return append(x, b...)
	}
	font := section("FILE", []byte("PFF2"))
	for _, s := range []struct {
		tag string
		b   []byte
	}{{"NAME", []byte("Fixture Regular 16\x00")}, {"CHIX", nil}, {"DATA", nil}} {
		font = append(font, section(s.tag, s.b)...)
	}
	testutil.Write(t, d, "font.pf2", font)
	testutil.Write(t, d, "theme.txt", []byte("title-font: \"Fixture Regular 16\"\n"))
	testutil.Write(t, d, "hd/theme.txt", []byte("title-text: \"HD\"\n"))
	m := recipe()
	m.Variants = []model.Variant{{ID: "hd", Root: "hd", Entry: "theme.txt"}}
	if v := Run(d, m); !v.Valid {
		t.Fatalf("%+v", v)
	}
	m.Variants = append(m.Variants, m.Variants[0])
	if Run(d, m).Valid {
		t.Fatal("duplicate variant accepted")
	}
}
