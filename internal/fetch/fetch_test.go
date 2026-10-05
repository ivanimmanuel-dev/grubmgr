package fetch

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"grubmgr/internal/model"
	"grubmgr/internal/state"
	"grubmgr/internal/testutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestBuiltinImmutable(t *testing.T) {
	p := testutil.Root(t, "debian")
	a, e := Import(p, "cyberpunk-demo", "")
	if e != nil || !a.Validation.Valid {
		t.Fatal(a, e)
	}
	b, e := Import(p, "cyberpunk-demo", "")
	if e != nil || a.Manifest.Revision != b.Manifest.Revision {
		t.Fatal(e)
	}
	db, e := state.Open(p, false)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	all, e := db.Packages()
	if e != nil || len(all) != 1 {
		t.Fatal(all, e)
	}
	testutil.Write(t, state.Content(p, a.Manifest.Revision), "theme.txt", []byte("changed"))
	if _, e = Import(p, "cyberpunk-demo", ""); e == nil {
		t.Fatal("cache tampering accepted")
	}
}
func TestLocalAndAmbiguity(t *testing.T) {
	p := testutil.Root(t, "debian")
	d := t.TempDir()
	testutil.Write(t, d, "theme.txt", []byte("title-text: \"local\"\n"))
	a, e := Import(p, d, "")
	if e != nil || !a.Validation.Valid || a.Manifest.License.Verified {
		t.Fatal(a, e)
	}
	testutil.Write(t, d, "nested/theme.txt", []byte("title-text: \"second\"\n"))
	if _, e = Import(p, d, ""); e == nil {
		t.Fatal("ambiguous roots accepted")
	}
	if _, e = Import(p, "community/minegrub", ""); e == nil {
		t.Fatal("browse-only package imported")
	}
}
func TestArchiveAndRecipe(t *testing.T) {
	p := testutil.Root(t, "debian")
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	w, _ := z.Create("nested/theme.txt")
	w.Write([]byte("title-text: \"zip\"\n"))
	z.Close()
	file := filepath.Join(t.TempDir(), "theme.zip")
	os.WriteFile(file, b.Bytes(), 0600)
	pkg, e := Import(p, file, "")
	if e != nil || pkg.Manifest.ArtifactSHA256 != model.Hash(b.Bytes()) || pkg.Manifest.Root != "nested" {
		t.Fatal(pkg, e)
	}
	recipe := pkg.Manifest.Recipe
	recipe.ArtifactSHA256 = model.Hash([]byte("wrong"))
	data, _ := json.Marshal(recipe)
	recipeFile := filepath.Join(t.TempDir(), "recipe.json")
	os.WriteFile(recipeFile, data, 0600)
	if _, e = Import(p, file, recipeFile); e == nil {
		t.Fatal("hash mismatch accepted")
	}
	os.WriteFile(recipeFile, []byte(`{"schema_version":1,"hook":"sh"}`), 0600)
	if _, e = ReadRecipe(recipeFile); e == nil {
		t.Fatal("hook accepted")
	}
}
func TestDownloadPolicy(t *testing.T) {
	if e := Download("http://example.invalid/theme.zip", filepath.Join(t.TempDir(), "a.zip"), model.Hash(nil)); e == nil {
		t.Fatal("plaintext accepted")
	}
}
func TestHTTPSArtifact(t *testing.T) {
	payload := []byte("bounded fixture artifact")
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "http://example.invalid/theme.zip", 302)
			return
		}
		w.Write(payload)
	}))
	defer s.Close()
	for _, tt := range []struct {
		path, digest string
		ok           bool
	}{{"/artifact", model.Hash(payload), true}, {"/artifact", model.Hash(nil), false}, {"/redirect", model.Hash(payload), false}} {
		file := filepath.Join(t.TempDir(), "artifact.zip")
		e := downloadWithClient(s.URL+tt.path, file, tt.digest, s.Client())
		if (e == nil) != tt.ok {
			t.Fatal(tt, e)
		}
	}
}
