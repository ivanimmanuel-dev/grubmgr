package debian

import (
	"encoding/base64"
	"encoding/json"
	"grubmgr/internal/catalog"
	"grubmgr/internal/model"
	"sort"
	"strings"
	"testing"
)

func demoBundle(t *testing.T) *Bundle {
	t.Helper()
	entry, err := catalog.Find("cyberpunk-demo")
	if err != nil {
		t.Fatal(err)
	}
	b := &Bundle{Manifest: model.Manifest{Recipe: entry.Recipe}, Files: catalog.DemoFiles()}
	for name, data := range b.Files {
		b.Manifest.Files = append(b.Manifest.Files, model.File{Path: name, Size: int64(len(data)), SHA256: model.Hash(data)})
	}
	sort.Slice(b.Manifest.Files, func(i, j int) bool { return b.Manifest.Files[i].Path < b.Manifest.Files[j].Path })
	b.Manifest.TreeSHA256 = model.Digest(b.Manifest.Files)
	b.Manifest.ArtifactSHA256 = b.Manifest.TreeSHA256
	b.Manifest.Revision = b.Manifest.Identity()
	return b
}

func TestRequestContract(t *testing.T) {
	for _, body := range []string{`{"operation":"shell"}`, `{"operation":"inspect","destination":"/boot"}`, `{"operation":"inspect","plan":"x"}`, `{"operation":"apply","action":{"action":"switch"}}`, `{"operation":"recover"} {}`, `{"operation":"plan","command":"true"}`} {
		if _, err := ReadRequest(strings.NewReader(body)); err == nil {
			t.Fatal("accepted invalid request", body)
		}
	}
	if _, err := ReadRequest(strings.NewReader(`{"operation":"inspect"}`)); err != nil {
		t.Fatal(err)
	}
}

func TestBundlePinsContent(t *testing.T) {
	b := demoBundle(t)
	if err := CheckBundle(b); err != nil {
		t.Fatal(err)
	}
	b.Files["theme.txt"] = []byte("title-text: \"changed\"\n")
	if err := CheckBundle(b); err == nil {
		t.Fatal("accepted altered content")
	}
	for i := range b.Manifest.Files {
		f := &b.Manifest.Files[i]
		data := b.Files[f.Path]
		f.Size = int64(len(data))
		f.SHA256 = model.Hash(data)
	}
	b.Manifest.TreeSHA256 = model.Digest(b.Manifest.Files)
	b.Manifest.Revision = b.Manifest.Identity()
	if err := CheckBundle(b); err == nil {
		t.Fatal("accepted recomputed caller hashes for a different builtin theme")
	}
}

func TestSettingsPreserveUnrelatedBytes(t *testing.T) {
	before := []byte("# Debian defaults\nGRUB_DISTRIBUTOR=`lsb_release -i -s`\nGRUB_DEFAULT=2\nGRUB_TIMEOUT=7\nGRUB_CMDLINE_LINUX=\"quiet\"\nGRUB_DISABLE_OS_PROBER=false\n")
	theme := "/boot/grub/themes/grubmgr/test/theme/abc/theme.txt"
	after, err := settings(before, theme)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before)+"GRUB_THEME=\""+theme+"\"\n" {
		t.Fatal(string(after))
	}
	cleared, err := settings(after, "")
	if err != nil || string(cleared) != string(before) {
		t.Fatal(string(cleared), err)
	}
	for _, b := range []string{"GRUB_THEME=$(x)\n", "GRUB_THEME=/a\nGRUB_THEME=/b\n"} {
		if _, err := settings([]byte(b), theme); err == nil {
			t.Fatal("accepted ambiguous theme")
		}
	}
}

func TestTokenRejectsTrailingData(t *testing.T) {
	b, _ := json.Marshal(Token{Schema: 2, Fingerprint: strings.Repeat("a", 64)})
	if _, err := DecodeToken("p2." + base64.RawURLEncoding.EncodeToString(append(b, []byte(" {}")...))); err == nil {
		t.Fatal("accepted trailing data")
	}
}
