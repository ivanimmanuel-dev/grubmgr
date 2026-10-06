package debian

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/output"
	"grubmgr/internal/planner"
	"grubmgr/internal/state"
	"os"
	"path"
	"sort"
)

func build(root string, req planner.Request, bundle *Bundle) (planner.Plan, error) {
	pl := planner.Plan{Request: req, Add: []string{}, Remove: []string{}, Retained: []string{}, Privileges: "Polkit authorizes grubmgr-helper", Recovery: "Immediate recovery restores verified snapshots; later rollback regenerates with current kernels"}
	if req.Variant == "" {
		req.Variant = "default"
		pl.Request = req
	}
	switch req.Action {
	case "install", "switch", "remove", "rollback":
	default:
		return pl, fmt.Errorf("unknown plan action")
	}
	env, err := inspect(root)
	if err != nil {
		return pl, err
	}
	if env.Status != "SUPPORTED" && env.Status != "SUPPORTED WITH WARNINGS" {
		return pl, output.Fail(output.Unsupported, "BACKEND_REFUSED", "%s", env.Reason)
	}
	if err = CheckBundle(bundle); err != nil {
		return pl, err
	}
	p := paths(root)
	if err = checkStore(); err != nil {
		return pl, err
	}
	db, err := state.Open(p, false)
	if err != nil {
		return pl, err
	}
	defer db.Close()
	packages, err := db.Packages()
	if err != nil {
		return pl, err
	}
	history, err := db.History()
	if err != nil {
		return pl, err
	}
	for _, tx := range history {
		if tx.Phase != "committed" && tx.Phase != "rolled_back" {
			return pl, output.Fail(output.Recovery, "RECOVERY_REQUIRED", "recover transaction %s first", tx.ID)
		}
	}
	if bundle != nil {
		found := false
		for _, pkg := range packages {
			if pkg.Manifest.Revision == bundle.Manifest.Revision {
				found = true
			}
		}
		if !found {
			packages = append(packages, model.Package{Manifest: bundle.Manifest, Variant: "default", Validation: model.Validation{Valid: true, Validator: "grubmgr-helper/package-identity", Findings: []model.Finding{}}})
		}
	}
	sort.Slice(packages, func(i, j int) bool { return packages[i].Manifest.Revision < packages[j].Manifest.Revision })
	r, err := os.OpenRoot(root)
	if err != nil {
		return pl, err
	}
	defer r.Close()
	defaults, err := fsx.Read(r, Defaults, 1<<20)
	if err != nil {
		return pl, err
	}
	cfg, err := fsx.Read(r, Config, 8<<20)
	if err != nil {
		return pl, err
	}
	beforeTheme, _ := themeValue(defaults)
	packages, err = activePackages(packages, beforeTheme)
	if err != nil {
		return pl, err
	}
	pl.Before = state.Snapshot{Defaults: defaults, Config: cfg, Packages: packages}
	pl.DefaultsBefore = string(defaults)
	pl.DefaultsAfter = string(defaults)
	pl.BeforeTheme, _ = themeValue(defaults)
	pl.AfterTheme = pl.BeforeTheme
	var selected model.Package
	have := true
	if req.Action == "rollback" {
		var prior *state.Transaction
		for i := range history {
			if history[i].ID == req.Target && history[i].Phase == "committed" {
				prior = &history[i]
			}
		}
		if prior == nil {
			return pl, fmt.Errorf("committed rollback target not found")
		}
		have = false
		pl.AfterTheme = ""
		for _, pkg := range prior.Before.Packages {
			if pkg.Active {
				selected, err = state.Select(packages, pkg.Manifest.Revision)
				if err != nil {
					return pl, err
				}
				pl.Request.Variant = pkg.Variant
				have = true
				break
			}
		}
		if !have {
			old, _ := themeValue(prior.Before.Defaults)
			if old != "" {
				if prior.Before.OriginalTheme == nil {
					return pl, fmt.Errorf("this older transaction has no previous theme backup")
				}
				if err := verifyOriginal(*prior.Before.OriginalTheme); err != nil {
					return pl, err
				}
				pl.AfterTheme = path.Join(originalDestination(*prior.Before.OriginalTheme), prior.Before.OriginalTheme.Entry)
			}
		}
	} else {
		selected, err = state.Select(packages, req.Target)
		if err != nil {
			return pl, err
		}
	}
	if have {
		m := selected.Manifest
		compatible := len(m.Compatibility.Backends) == 0
		for _, backend := range m.Compatibility.Backends {
			if compatibleBackend(backend, env.Backend) {
				compatible = true
			}
		}
		if !compatible {
			return pl, fmt.Errorf("recipe does not permit %s", env.Backend)
		}
		for _, constraint := range []struct {
			allowed []string
			actual  string
		}{
			{m.Compatibility.Architectures, "amd64"}, {m.Compatibility.Firmware, "uefi"},
		} {
			if len(constraint.allowed) == 0 {
				continue
			}
			matched := false
			for _, value := range constraint.allowed {
				if value == constraint.actual {
					matched = true
				}
			}
			if !matched {
				return pl, fmt.Errorf("theme does not support %s", constraint.actual)
			}
		}
		if bundle != nil && bundle.Manifest.Revision != m.Revision {
			return pl, fmt.Errorf("supplied package does not match plan target")
		}
		if bundle == nil {
			if err = protectedDestination(state.Content(p, m.Revision)); err != nil {
				return pl, err
			}
			if _, digest, err := fsx.Inventory(state.Content(p, m.Revision)); err != nil || digest != m.TreeSHA256 {
				return pl, fmt.Errorf("root package cache failed verification")
			}
		}
		variant, err := m.Variant(pl.Request.Variant)
		if err != nil {
			return pl, err
		}
		if (variant.Root != "." && fsx.SafePath(variant.Root) != nil) || fsx.SafePath(variant.Entry) != nil {
			return pl, fmt.Errorf("invalid theme entry")
		}
		pl.Package = &selected
		pl.Revision = m.Revision
		pl.ThemeID = m.ID
		pl.TreeSHA256 = m.TreeSHA256
		pl.ArtifactSHA256 = m.ArtifactSHA256
		pl.Destination = "/boot/grub/themes/grubmgr/" + m.ID + "/" + m.Revision
		if err = protectedDestination(pl.Destination); err != nil {
			return pl, err
		}
		if req.Action == "switch" || req.Action == "remove" {
			if !selected.Installed {
				return pl, fmt.Errorf("install this revision first")
			}
		}
		if req.Action == "switch" || req.Action == "rollback" {
			pl.AfterTheme = path.Join(pl.Destination, variant.Root, variant.Entry)
		}
		if req.Action == "remove" && selected.Active {
			pl.AfterTheme = ""
		}
		if req.Action == "install" || req.Action == "rollback" {
			for _, file := range m.Files {
				pl.Add = append(pl.Add, path.Join(pl.Destination, file.Path))
			}
		}
		pl.Retained = append(pl.Retained, pl.Destination)
	}
	if pl.AfterTheme != pl.BeforeTheme {
		managed := false
		for _, pkg := range packages {
			if !pkg.Installed {
				continue
			}
			v, err := pkg.Manifest.Variant(pkg.Variant)
			if err != nil {
				return pl, err
			}
			if path.Join("/boot/grub/themes/grubmgr", pkg.Manifest.ID, pkg.Manifest.Revision, v.Root, v.Entry) == pl.BeforeTheme {
				managed = true
			}
		}
		if pl.BeforeTheme != "" && !managed {
			pl.Before.OriginalTheme, err = originalSnapshot(root, pl.BeforeTheme)
			if err != nil {
				return pl, err
			}
			pl.ThemeBackup = originalDestination(*pl.Before.OriginalTheme)
		}
		after, err := settings(defaults, pl.AfterTheme)
		if err != nil {
			return pl, err
		}
		pl.DefaultsAfter = string(after)
	}
	if req.Action != "install" && (pl.AfterTheme != pl.BeforeTheme || req.Action == "rollback") {
		pl.Generator = []string{"/" + env.profile.Generator, "-o", "/boot/grub/.grubmgr-TRANSACTION.cfg"}
	}
	pl.Fingerprint = model.Digest(struct {
		Environment string
		Before      state.Snapshot
		History     []state.Transaction
	}{env.Fingerprint, pl.Before, history})
	b, _ := json.Marshal(Token{Schema: 2, Request: pl.Request, Fingerprint: pl.Fingerprint, Revision: pl.Revision})
	pl.ID = "p2." + base64.RawURLEncoding.EncodeToString(b)
	pl.Applicable = true
	pl.Compatibility = env.Backend
	pl.Validation = "package identity verified; candidate checked during apply"
	pl.Reason = env.Reason
	return pl, nil
}

func activePackages(packages []model.Package, theme string) ([]model.Package, error) {
	for i := range packages {
		pkg := &packages[i]
		variant, err := pkg.Manifest.Variant(pkg.Variant)
		if err != nil {
			return nil, err
		}
		pkg.Active = pkg.Installed && path.Join("/boot/grub/themes/grubmgr", pkg.Manifest.ID, pkg.Manifest.Revision, variant.Root, variant.Entry) == theme
	}
	return packages, nil
}
