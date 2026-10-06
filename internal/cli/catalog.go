package cli

import (
	"fmt"
	"grubmgr/internal/catalog"
	"grubmgr/internal/fetch"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/output"
	"grubmgr/internal/system"
	"os"
	"path/filepath"
	"strings"
)

func configureCatalog(o options, p system.Paths) (any, error) {
	a := o.args[1:]
	if len(a) == 1 && a[0] == "list" {
		return catalog.List(p.Config)
	}
	if len(a) == 2 && a[0] == "remove" {
		err := catalog.Remove(p.Config, a[1])
		return fmt.Sprintf("Removed catalog %s\n", a[1]), err
	}
	if len(a) != 3 || a[0] != "add" {
		return nil, output.Fail(output.Usage, "ARGUMENT", "use catalog list, catalog add NAME SOURCE [--sha256 HASH], or catalog remove NAME")
	}
	source := a[2]
	if strings.HasPrefix(source, "https://") || strings.HasPrefix(source, "http://") {
		if !model.IsDigest(o.digest) {
			return nil, output.Fail(output.Usage, "CATALOG_DIGEST", "an online catalog requires --sha256 with its exact SHA-256")
		}
		if err := p.Ensure(); err != nil {
			return nil, err
		}
		file, err := os.CreateTemp(p.Cache, "catalog-*.json")
		if err != nil {
			return nil, err
		}
		file.Close()
		defer os.Remove(file.Name())
		if err = fetch.Download(source, file.Name(), o.digest); err != nil {
			return nil, err
		}
		source = file.Name()
	}
	file, err := filepath.Abs(source)
	if err != nil {
		return nil, err
	}
	if err = fsx.NoLinks(file); err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(filepath.Dir(file))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	data, err := fsx.Read(r, filepath.Base(file), catalog.MaxRegistry)
	if err != nil {
		return nil, err
	}
	if o.digest != "" && (!model.IsDigest(o.digest) || model.Hash(data) != o.digest) {
		return nil, fmt.Errorf("catalog digest does not match")
	}
	return catalog.Add(p.Config, a[1], data)
}
