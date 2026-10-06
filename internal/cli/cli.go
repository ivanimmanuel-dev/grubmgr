package cli

import (
	"fmt"
	"grubmgr/internal/catalog"
	"grubmgr/internal/debian"
	"grubmgr/internal/fetch"
	"grubmgr/internal/model"
	"grubmgr/internal/output"
	"grubmgr/internal/planner"
	"grubmgr/internal/preview"
	"grubmgr/internal/state"
	"grubmgr/internal/system"
	"grubmgr/internal/transaction"
	"grubmgr/internal/validate"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
)

const Version = "0.4.0-rc.1"
const help = `GRUB Manager (grubmgr)

Usage: grubmgr [--json] [--log-json] [--root FIXTURE] COMMAND

  version | help
  doctor                            Read-only system inspection
  search [QUERY] | info ID           Browse themes
  catalog list | add NAME SOURCE [--sha256 HASH] | remove NAME
  fetch SOURCE [--recipe FILE]       Import a theme package
  validate PATH_OR_ID [--recipe FILE]
  list                              Show imported packages
  status | history                  Show managed state and transactions
  install SOURCE [--recipe FILE]    Import and install a theme
  switch ID [--variant ID]          Select an installed theme
  remove ID | rollback TRANSACTION_ID
  plan install|switch|remove ID [--variant ID]
  plan rollback TRANSACTION_ID       Restore the previous theme selection
  apply TOKEN                       Apply the reviewed plan
  recover                           Recover an interrupted transaction
  preview ID [--variant ID]          Render a menu with GRUB and QEMU

Activation supports Debian 13, Ubuntu 24.04, Kali and Arch UEFI installations.
Install copies assets; switch activates a theme.
Use --yes to confirm a direct operation in scripts.
`

type options struct {
	json                          bool
	logJSON                       bool
	yes                           bool
	root, recipe, variant, digest string
	args                          []string
}

func parse(args []string) (options, error) {
	o := options{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--json":
			o.json = true
		case "--log-json":
			o.logJSON = true
		case "--yes", "-y":
			o.yes = true
		case "--root", "--recipe", "--variant", "--sha256":
			if i+1 == len(args) {
				return o, output.Fail(output.Usage, "ARGUMENT", "%s requires a value", a)
			}
			i++
			switch a {
			case "--root":
				o.root = args[i]
			case "--recipe":
				o.recipe = args[i]
			case "--variant":
				o.variant = args[i]
			case "--sha256":
				o.digest = args[i]
			}
		case "--help", "-h":
			o.args = []string{"help"}
			return o, nil
		case "--version":
			o.args = []string{"version"}
			return o, nil
		default:
			if strings.HasPrefix(a, "-") {
				return o, output.Fail(output.Usage, "ARGUMENT", "unknown option %s", a)
			}
			o.args = append(o.args, a)
		}
	}
	if len(o.args) == 0 {
		o.args = []string{"help"}
	}
	return o, nil
}
func Run(args []string, stdout, stderr io.Writer) (code int) {
	return RunWithInput(args, os.Stdin, stdout, stderr)
}

func RunWithInput(args []string, stdin io.Reader, stdout, stderr io.Writer) (code int) {
	o, e := parse(args)
	if e != nil {
		return output.Report(stderr, e, o.json)
	}
	if o.logJSON {
		logger := slog.New(slog.NewJSONHandler(stderr, nil))
		logger.Info("command started", "command", o.args[0])
		defer func() { logger.Info("command finished", "command", o.args[0], "exit_code", code) }()
	}
	var value any
	switch o.args[0] {
	case "install", "switch", "remove", "rollback":
		value, e = direct(o, stdin, stdout, stderr)
	default:
		value, e = execute(o)
	}
	if e != nil {
		return output.Report(stderr, e, o.json)
	}
	if o.json {
		if e = output.Write(stdout, value); e != nil {
			return output.Report(stderr, e, true)
		}
	} else {
		render(stdout, value)
	}
	if v, ok := value.(model.Validation); ok && !v.Valid {
		return output.Invalid
	}
	if p, ok := value.(model.Package); ok && !p.Validation.Valid {
		return output.Invalid
	}
	return 0
}
func execute(o options) (any, error) {
	command := o.args[0]
	a := o.args[1:]
	need := func(n int) error {
		if len(a) != n {
			return output.Fail(output.Usage, "ARGUMENT", "%s expects %d arguments", command, n)
		}
		return nil
	}
	if command == "help" {
		return help, need(0)
	}
	if command == "version" {
		return map[string]any{"version": Version, "go": runtime.Version(), "activation": "Debian 13, Ubuntu 24.04, Kali and Arch UEFI installations"}, need(0)
	}
	p, e := system.Locations(o.root)
	if e != nil {
		return nil, e
	}
	if !p.Fixture && debian.Available() {
		switch command {
		case "doctor":
			if e = need(0); e != nil {
				return nil, e
			}
			var report debian.Environment
			e = debian.Call(debian.Request{Operation: "inspect"}, &report)
			return report, e
		case "plan":
			if e = need(2); e != nil {
				return nil, e
			}
			req := debian.Request{Operation: "plan", Action: planner.Request{Action: a[0], Target: canonical(a[1], p.Config), Variant: o.variant}}
			if a[0] == "rollback" {
				req.Action.Target = a[1]
			}
			if a[0] == "install" {
				req.Package, e = debian.LoadBundle(p, req.Action.Target)
				if e != nil {
					return nil, e
				}
				req.Action.Target = req.Package.Manifest.Revision
			}
			var plan planner.Plan
			e = debian.Call(req, &plan)
			return plan, e
		case "apply":
			if e = need(1); e != nil {
				return nil, e
			}
			token, err := debian.DecodeToken(a[0])
			if err != nil {
				return nil, err
			}
			req := debian.Request{Operation: "apply", Plan: a[0]}
			if token.Request.Action == "install" {
				req.Package, e = debian.LoadBundle(p, token.Revision)
				if e != nil {
					return nil, e
				}
			}
			var tx state.Transaction
			e = debian.Call(req, &tx)
			if e != nil {
				return nil, e
			}
			return map[string]any{"transaction": tx.ID, "phase": tx.Phase, "events": tx.Events}, nil
		case "recover", "history", "status":
			if e = need(0); e != nil {
				return nil, e
			}
			if command == "status" {
				var packages []model.Package
				e = debian.Call(debian.Request{Operation: command}, &packages)
				return packages, e
			}
			var history []state.Transaction
			e = debian.Call(debian.Request{Operation: command}, &history)
			return history, e
		}
	}
	switch command {
	case "catalog":
		return configureCatalog(o, p)
	case "doctor":
		if e = need(0); e != nil {
			return nil, e
		}
		return system.Inspect(p)
	case "search":
		if len(a) > 1 {
			return nil, output.Fail(output.Usage, "ARGUMENT", "search takes zero or one query")
		}
		q := ""
		if len(a) == 1 {
			q = a[0]
		}
		return catalog.SearchConfigured(p.Config, q)
	case "info":
		if e = need(1); e != nil {
			return nil, e
		}
		db, e := state.Open(p, false)
		if e != nil {
			return nil, e
		}
		defer db.Close()
		packages, e := db.Packages()
		if e != nil {
			return nil, e
		}
		if pkg, err := state.Select(packages, a[0]); err == nil {
			return pkg, nil
		}
		return catalog.FindConfigured(p.Config, a[0])
	case "fetch":
		if e = need(1); e != nil {
			return nil, e
		}
		if !p.Fixture && runtime.GOOS != "windows" && os.Geteuid() == 0 {
			return nil, output.Fail(output.Unsupported, "UNPRIVILEGED_REQUIRED", "fetch must run as an ordinary user")
		}
		return fetch.Import(p, a[0], o.recipe)
	case "validate":
		if e = need(1); e != nil {
			return nil, e
		}
		dir := a[0]
		if i, err := os.Stat(dir); err == nil && i.IsDir() {
			var recipe model.Recipe
			if o.recipe != "" {
				recipe, e = fetch.ReadRecipe(o.recipe)
			} else {
				recipe, e = fetch.Infer(dir, dir)
			}
			if e != nil {
				return nil, e
			}
			return validate.Run(dir, model.Manifest{Recipe: recipe}), nil
		}
		db, e := state.Open(p, false)
		if e != nil {
			return nil, e
		}
		defer db.Close()
		packages, e := db.Packages()
		if e != nil {
			return nil, e
		}
		pkg, e := state.Select(packages, canonical(a[0], p.Config))
		if e != nil {
			return nil, output.Fail(output.Invalid, "PACKAGE_SELECTION", "%v", e)
		}
		return validate.Run(state.Content(p, pkg.Manifest.Revision), pkg.Manifest), nil
	case "list", "status", "history":
		if e = need(0); e != nil {
			return nil, e
		}
		db, e := state.Open(p, false)
		if e != nil {
			return nil, e
		}
		defer db.Close()
		if command == "history" {
			history, e := db.History()
			if e != nil {
				return nil, e
			}
			view := []map[string]any{}
			for _, t := range history {
				view = append(view, map[string]any{"id": t.ID, "created_at": t.CreatedAt, "action": t.Action, "revision": t.Revision, "phase": t.Phase, "events": t.Events, "error": t.Error})
			}
			return view, nil
		}
		packages, e := db.Packages()
		if e != nil {
			return nil, e
		}
		if command == "list" {
			return packages, nil
		}
		report, e := system.Inspect(p)
		if e != nil {
			return nil, e
		}
		return map[string]any{"system": report, "packages": packages, "state_path": p.State, "data_path": p.Data, "config_path": p.Config, "cache_path": p.Cache, "real_activation": false}, nil
	case "plan":
		if e = need(2); e != nil {
			return nil, e
		}
		target := a[1]
		if a[0] != "rollback" {
			target = canonical(target, p.Config)
		}
		return planner.Build(p, planner.Request{Action: a[0], Target: target, Variant: o.variant})
	case "apply":
		if e = need(1); e != nil {
			return nil, e
		}
		t, e := transaction.Apply(p, a[0], transaction.Faults{})
		if e != nil {
			return nil, e
		}
		return map[string]any{"transaction": t.ID, "phase": t.Phase, "events": t.Events, "fixture_root": p.Root}, nil
	case "recover":
		if e = need(0); e != nil {
			return nil, e
		}
		return transaction.Recover(p)
	case "preview":
		if e = need(1); e != nil {
			return nil, e
		}
		db, e := state.Open(p, false)
		if e != nil {
			return nil, e
		}
		defer db.Close()
		packages, e := db.Packages()
		if e != nil {
			return nil, e
		}
		pkg, e := state.Select(packages, canonical(a[0], p.Config))
		if e != nil {
			return nil, e
		}
		if !p.Fixture {
			return preview.Capture(state.Content(p, pkg.Manifest.Revision), filepath.Join(p.Cache, "previews"), pkg.Manifest, o.variant)
		}
		report, e := system.Inspect(p)
		if e != nil {
			return nil, e
		}
		return nil, (preview.External{System: report}).Preview(filepath.Clean(state.Content(p, pkg.Manifest.Revision)))
	default:
		return nil, output.Fail(output.Usage, "COMMAND", "unknown command %s; see help", command)
	}
}
func canonical(id string, config ...string) string {
	var entry catalog.Entry
	var e error
	if len(config) == 0 {
		entry, e = catalog.Find(id)
	} else {
		entry, e = catalog.FindConfigured(config[0], id)
	}
	if e == nil {
		return entry.Recipe.ID
	}
	return id
}
func render(w io.Writer, value any) {
	switch v := value.(type) {
	case debian.Environment:
		fmt.Fprintf(w, "Support: %s\nBackend: %s\nGRUB: %s\nFirmware: %s\nLayout: %s\n%s\n", v.Status, v.Backend, v.GRUBVersion, v.Firmware, v.Layout, v.Reason)
	case string:
		fmt.Fprint(w, v)
	case system.Report:
		fmt.Fprintf(w, "GRUB Manager %s\n\nSystem: %s %s / %s / %s\nGRUB evidence: %t; version: %s\nTheme: %s\nBackend: %s (%s)\n%s\nPreview: %s\n", Version, v.Distribution, v.Version, v.Architecture, v.Firmware, v.GRUBInstalled, v.GRUBVersion, v.Theme, v.Backend, v.Status, v.Reason, v.Preview)
		for _, s := range v.Warnings {
			fmt.Fprintln(w, "WARN", s)
		}
	case model.Validation:
		for _, f := range v.Findings {
			fmt.Fprintf(w, "%s %s %s:%d — %s\n", f.Severity, f.Code, f.File, f.Line, f.Message)
		}
		fmt.Fprintln(w, validate.Summary(v))
	case model.Package:
		fmt.Fprintf(w, "%s\nID: %s\nRevision: %s\nSource: %s\nLicense: %s\nValidation: %s\n", v.Manifest.Name, v.Manifest.ID, v.Manifest.Revision, v.Manifest.Source.URL, v.Manifest.License.SPDX, validate.Summary(v.Validation))
	case []model.Package:
		table := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		fmt.Fprintln(table, "ID\tREVISION\tINSTALLED\tACTIVE\tVARIANT")
		for _, pkg := range v {
			fmt.Fprintf(table, "%s\t%.12s\t%t\t%t\t%s\n", pkg.Manifest.ID, pkg.Manifest.Revision, pkg.Installed, pkg.Active, pkg.Variant)
		}
		table.Flush()
	case []catalog.Entry:
		table := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		fmt.Fprintln(table, "ID\tNAME\tSOURCE")
		for _, entry := range v {
			fmt.Fprintf(table, "%s\t%s\t%s\n", entry.Recipe.ID, entry.Recipe.Name, entry.Status)
		}
		table.Flush()
	case catalog.Entry:
		fmt.Fprintf(w, "%s\nID: %s\nVersion: %s\nAuthor: %s\nSource: %s\nLicense: %s\n", v.Recipe.Name, v.Recipe.ID, v.Recipe.Version, v.Recipe.Author, v.Recipe.Source.URL, v.Recipe.License.SPDX)
	case catalog.Registry:
		fmt.Fprintf(w, "Added catalog %s: %d themes\nSHA-256: %s\n", v.Name, v.Themes, v.SHA256)
	case []catalog.Registry:
		for _, registry := range v {
			fmt.Fprintf(w, "%s\t%d themes\t%s\n", registry.Name, registry.Themes, registry.SHA256)
		}
	case []state.Transaction:
		table := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		fmt.Fprintln(table, "TRANSACTION\tACTION\tRESULT\tTIME")
		for _, tx := range v {
			fmt.Fprintf(table, "%s\t%s\t%s\t%s\n", tx.ID, tx.Action, tx.Phase, tx.CreatedAt)
		}
		table.Flush()
	case map[string]any:
		if version, ok := v["version"]; ok {
			fmt.Fprintf(w, "GRUB Manager %s\n", version)
		} else if transaction, ok := v["transaction"]; ok {
			fmt.Fprintf(w, "Transaction %s: %s\n", transaction, v["phase"])
		} else {
			_ = output.Write(w, value)
		}
	case planner.Plan:
		fmt.Fprintf(w, "PLAN %s\n\n%s %s@%s\nArtifact SHA-256: %s\nTree SHA-256: %s\nDestination: %s\nTheme: %q -> %q\nValidation: %s\nCompatibility: %s\nPrivileges: %s\nGenerator: %v\nApplicable: %t — %s\n", v.ID, v.Request.Action, v.ThemeID, v.Revision, v.ArtifactSHA256, v.TreeSHA256, v.Destination, v.BeforeTheme, v.AfterTheme, v.Validation, v.Compatibility, v.Privileges, v.Generator, v.Applicable, v.Reason)
		for _, s := range v.Add {
			fmt.Fprintln(w, "+", s)
		}
		for _, s := range v.Remove {
			fmt.Fprintln(w, "-", s)
		}
		for _, s := range v.Retained {
			fmt.Fprintln(w, "Retain:", s)
		}
		fmt.Fprintf(w, "Recovery: %s\n", v.Recovery)
	default:
		_ = output.Write(w, value)
	}
}
