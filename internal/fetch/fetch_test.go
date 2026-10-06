package fetch

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"grubmgr/internal/catalog"
	"grubmgr/internal/model"
	"grubmgr/internal/state"
	"grubmgr/internal/testutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	if _, err := Infer(d, "theme\nforged-output.zip"); err == nil {
		t.Fatal("unsafe inferred display metadata accepted")
	}
	testutil.Write(t, d, "nested/theme.txt", []byte("title-text: \"second\"\n"))
	if _, e = Import(p, d, ""); e == nil {
		t.Fatal("ambiguous roots accepted")
	}
}
func TestRemoteImportRequiresRecipe(t *testing.T) {
	p := testutil.Root(t, "debian")
	if _, err := Import(p, "https://example.invalid/theme.zip", ""); err == nil || !strings.Contains(err.Error(), "RECIPE_REQUIRED") {
		t.Fatal("expected a recipe requirement before download", err)
	}
}
func TestRecipeSelectsAssetsAndPreservesNotices(t *testing.T) {
	p := testutil.Root(t, "debian")
	directory := t.TempDir()
	testutil.Write(t, directory, "collection/theme/theme.txt", []byte("title-text: \"Community theme\"\n"))
	notice := []byte("Author's original license notice\n")
	testutil.Write(t, directory, "collection/LICENSE", notice)
	testutil.Write(t, directory, "collection/install.sh", []byte("#!/bin/sh\nexit 1\n"))
	recipe := model.Recipe{Schema: 1, ID: "community/example", Name: "Example", Root: "collection/theme", Entry: "theme.txt", Include: []string{"collection/theme"}, RecipeRevision: "1", License: model.License{Notices: []string{"collection/LICENSE"}}}
	recipeFile := filepath.Join(t.TempDir(), "recipe.json")
	writeRecipe := func() { data, _ := json.Marshal(recipe); os.WriteFile(recipeFile, data, 0600) }
	writeRecipe()
	pkg, err := Import(p, directory, recipeFile)
	if err != nil || !pkg.Validation.Valid || len(pkg.Manifest.Files) != 2 {
		t.Fatal(pkg, err)
	}
	content := state.Content(p, pkg.Manifest.Revision)
	got, err := os.ReadFile(filepath.Join(content, "collection/LICENSE"))
	if err != nil || !bytes.Equal(got, notice) {
		t.Fatal("notice changed", err)
	}
	if _, err := os.Stat(filepath.Join(content, "collection/install.sh")); !os.IsNotExist(err) {
		t.Fatal("installer retained", err)
	}
	for _, include := range [][]string{{"../outside"}, {"missing"}} {
		recipe.Include = include
		writeRecipe()
		if _, err := Import(p, directory, recipeFile); err == nil {
			t.Fatal("invalid selection accepted", include)
		}
	}
	recipe.Include = []string{"collection"}
	writeRecipe()
	pkg, err = Import(p, directory, recipeFile)
	if err != nil || pkg.Validation.Valid {
		t.Fatal("selected installer must fail validation", pkg, err)
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

func TestConfiguredCatalogImportsPinnedCollection(t *testing.T) {
	p := testutil.Root(t, "debian")
	var data bytes.Buffer
	z := zip.NewWriter(&data)
	for name, content := range map[string]string{"collection/theme/theme.txt": "title-text: \"Catalog theme\"\n", "collection/LICENSE": "Author's license\n", "collection/install.sh": "#!/bin/sh\nexit 1\n"} {
		w, _ := z.Create(name)
		w.Write([]byte(content))
	}
	z.Close()
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data.Bytes()) }))
	defer s.Close()
	recipe := model.Recipe{Schema: 1, ID: "author/catalog-theme", Name: "Catalog theme", Author: "Author", Source: model.Source{Provider: "https", URL: s.URL + "/theme.zip", UpstreamRevision: "v1"}, ArtifactSHA256: model.Hash(data.Bytes()), Root: "collection/theme", Entry: "theme.txt", Include: []string{"collection/theme"}, RecipeRevision: "1", Reviewed: true, License: model.License{Notices: []string{"collection/LICENSE"}}}
	index, _ := json.Marshal([]catalog.Entry{{Recipe: recipe}})
	if _, err := catalog.Add(p.Config, "collection", index); err != nil {
		t.Fatal(err)
	}
	pkg, err := importWithClient(p, recipe.ID, "", s.Client())
	if err != nil || !pkg.Validation.Valid || len(pkg.Manifest.Files) != 2 || pkg.Manifest.ArtifactSHA256 != recipe.ArtifactSHA256 || pkg.Manifest.Author != recipe.Author {
		t.Fatal(pkg, err)
	}
	// The same index must reject changed upstream bytes before importing them.
	data.WriteString("changed artifact")
	if _, err := importWithClient(p, recipe.ID, "", s.Client()); err == nil {
		t.Fatal("catalog artifact substitution accepted")
	}
}
