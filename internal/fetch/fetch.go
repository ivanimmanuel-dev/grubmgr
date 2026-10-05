package fetch

import (
	"encoding/json"
	"fmt"
	"grubmgr/internal/archive"
	"grubmgr/internal/catalog"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/output"
	"grubmgr/internal/state"
	"grubmgr/internal/system"
	"grubmgr/internal/validate"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func ReadRecipe(file string) (model.Recipe, error) {
	var r model.Recipe
	f, e := os.Open(file)
	if e != nil {
		return r, e
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 1<<20))
	d.DisallowUnknownFields()
	if e = d.Decode(&r); e != nil {
		return r, e
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return r, fmt.Errorf("recipe has trailing data")
	}
	return r, r.Check()
}
func Infer(dir, origin string) (model.Recipe, error) {
	files, _, e := fsx.Inventory(dir)
	if e != nil {
		return model.Recipe{}, e
	}
	roots := []string{}
	for _, f := range files {
		if filepath.Base(f.Path) == "theme.txt" {
			roots = append(roots, filepath.ToSlash(filepath.Dir(f.Path)))
		}
	}
	if len(roots) != 1 {
		return model.Recipe{}, output.Fail(output.Invalid, "AMBIGUOUS_THEME_ROOT", "found %d theme.txt files; supply an explicit --recipe", len(roots))
	}
	return model.Recipe{Schema: 1, ID: "local/import-" + model.Hash([]byte(origin))[:12], Name: "Local imported theme", Author: "unknown", Version: "local", Source: model.Source{Provider: "local", URL: origin, UpstreamRevision: "content-addressed"}, Root: roots[0], Entry: "theme.txt", RecipeRevision: "local-1", Preview: model.Preview{Kind: "none"}}, nil
}
func Download(raw, dest, expected string) error {
	return downloadWithClient(raw, dest, expected, &http.Client{Timeout: 60 * time.Second})
}
func downloadWithClient(raw, dest, expected string, base *http.Client) error {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || !model.IsDigest(expected) {
		return output.Fail(output.Invalid, "SOURCE_POLICY", "HTTPS source and reviewed SHA-256 are required")
	}
	client := *base
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || req.URL.Scheme != "https" || req.URL.User != nil {
			return fmt.Errorf("unsafe redirect")
		}
		return nil
	}
	res, e := client.Get(raw)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("download returned HTTP %d", res.StatusCode)
	}
	if res.ContentLength > fsx.MaxTree {
		return fmt.Errorf("download exceeds limit")
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, fsx.MaxTree+1))
	if e != nil {
		return e
	}
	if len(b) > int(fsx.MaxTree) || model.Hash(b) != expected {
		return output.Fail(output.Invalid, "ARTIFACT_HASH", "artifact exceeds limit or does not match the recipe digest")
	}
	return os.WriteFile(dest, b, 0600)
}
func Import(p system.Paths, source, recipeFile string) (model.Package, error) {
	var result model.Package
	var recipe model.Recipe
	var e error
	explicit := recipeFile != ""
	builtin := false
	if explicit {
		recipe, e = ReadRecipe(recipeFile)
		if e != nil {
			return result, e
		}
	} else if entry, err := catalog.Find(source); err == nil {
		if !entry.Recipe.Reviewed {
			return result, output.Fail(output.Unsupported, "BROWSE_ONLY", "%s has no reviewed package recipe", source)
		}
		recipe = entry.Recipe
		explicit = true
		source = recipe.Source.URL
		builtin = recipe.Source.Provider == "builtin"
	}
	remote := strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://")
	if remote && (!explicit || !recipe.Reviewed || recipe.Source.URL != source) {
		return result, output.Fail(output.Unsupported, "RECIPE_REQUIRED", "remote artifacts require an explicit reviewed recipe with an exact canonical URL")
	}
	if e = p.Ensure(); e != nil {
		return result, e
	}
	// Staging and the immutable store share a filesystem even when XDG cache and
	// data live on different mounts. Downloads remain unprivileged.
	temp, e := os.MkdirTemp(p.Data, ".stage-")
	if e != nil {
		return result, e
	}
	defer os.RemoveAll(temp)
	content := filepath.Join(temp, "content")
	if e = os.Mkdir(content, 0700); e != nil {
		return result, e
	}
	r, e := os.OpenRoot(content)
	if e != nil {
		return result, e
	}
	defer r.Close()
	artifact := ""
	switch {
	case builtin:
		for name, b := range catalog.DemoFiles() {
			if e = fsx.Write(r, name, b); e != nil {
				return result, e
			}
		}
	case remote:
		u, _ := url.Parse(source)
		artifactFile := filepath.Join(temp, filepath.Base(u.Path))
		if e = Download(source, artifactFile, recipe.ArtifactSHA256); e != nil {
			return result, e
		}
		artifact = recipe.ArtifactSHA256
		if e = archive.Extract(artifactFile, content, archive.Default); e != nil {
			return result, output.Fail(output.Invalid, "ARCHIVE_REJECTED", "%v", e)
		}
	default:
		abs, err := filepath.Abs(source)
		if err != nil {
			return result, err
		}
		if e = fsx.NoLinks(abs); e != nil {
			return result, e
		}
		i, err := os.Stat(abs)
		if err != nil {
			return result, err
		}
		if i.IsDir() {
			if e = fsx.CopyTree(abs, r, ""); e != nil {
				return result, e
			}
		} else {
			if i.Size() > fsx.MaxTree {
				return result, fmt.Errorf("artifact size limit")
			}
			b, err := os.ReadFile(abs)
			if err != nil {
				return result, err
			}
			artifact = model.Hash(b)
			if e = archive.Extract(abs, content, archive.Default); e != nil {
				return result, output.Fail(output.Invalid, "ARCHIVE_REJECTED", "%v", e)
			}
		}
		if !explicit {
			recipe, e = Infer(content, abs)
			if e != nil {
				return result, e
			}
		}
	}
	files, tree, e := fsx.Inventory(content)
	if e != nil {
		return result, e
	}
	if artifact == "" {
		artifact = tree
	}
	if recipe.ArtifactSHA256 != "" && recipe.ArtifactSHA256 != artifact {
		return result, output.Fail(output.Invalid, "ARTIFACT_HASH", "artifact does not match explicit recipe")
	}
	recipe.ArtifactSHA256 = artifact
	m := model.Manifest{Recipe: recipe, TreeSHA256: tree, Files: files}
	m.Revision = m.Identity()
	v := validate.Run(content, m)
	result = model.Package{Manifest: m, Validation: v, Variant: "default"}
	dest := filepath.Dir(state.Content(p, m.Revision))
	if e = fsx.NoLinks(dest); e != nil {
		return result, e
	}
	if e = os.MkdirAll(filepath.Dir(dest), 0700); e != nil {
		return result, e
	}
	r.Close()
	if _, e = os.Stat(dest); os.IsNotExist(e) {
		b, _ := json.MarshalIndent(m, "", "  ")
		if e = os.WriteFile(filepath.Join(temp, "manifest.json"), b, 0600); e != nil {
			return result, e
		}
		if e = os.Rename(temp, dest); e != nil {
			return result, e
		}
	} else if e != nil {
		return result, e
	} else {
		_, h, err := fsx.Inventory(state.Content(p, m.Revision))
		if err != nil || h != tree {
			return result, output.Fail(output.Conflict, "CACHE_CHANGED", "existing immutable cache failed verification")
		}
	}
	unlock, e := state.Lock(p)
	if e != nil {
		return result, output.Fail(output.Conflict, "LOCKED", "%v", e)
	}
	defer unlock()
	db, e := state.Open(p, true)
	if e != nil {
		return result, e
	}
	defer db.Close()
	packages, e := db.Packages()
	if e != nil {
		return result, e
	}
	for _, old := range packages {
		if old.Manifest.Revision == m.Revision {
			result.Installed = old.Installed
			result.Active = old.Active
			result.Pinned = old.Pinned
			result.Variant = old.Variant
		}
	}
	if e = db.Put(result); e != nil {
		return result, e
	}
	return result, nil
}
