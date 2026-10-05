package cli

import (
	"fmt"
	"grubmgr/internal/catalog"
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
)

const Version = "0.1.0"
const help = `GRUB Manager (grubmgr) 0.1.0 — community theme package manager

Usage: grubmgr [--json] [--log-json] [--root FIXTURE] COMMAND

  version | help
  doctor                            Read-only system inspection
  search [QUERY] | info ID           Browse shipped reviewed/discovery entries
  fetch SOURCE [--recipe FILE]       Import data into the immutable local store
  validate PATH_OR_ID [--recipe FILE]
  list | status | history           Read receipts and transaction history
  plan install|switch|remove ID [--variant ID]
  plan rollback TRANSACTION         Read-only plan; returns a p1.* token
  apply TOKEN                       Apply only to a marked Debian fixture root
  recover                           Recover interrupted fixture transactions
  preview ID                        Detect optional renderer; explicit refusal in 0.1

Real bootloader modification is disabled. No GRUB utility or theme script is run.
Install stages assets; switch activates an installed revision. Planning never writes.
`

type options struct {
	json                  bool
	logJSON               bool
	root, recipe, variant string
	args                  []string
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
		case "--root", "--recipe", "--variant":
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
	o, e := parse(args)
	if e != nil {
		return output.Report(stderr, e, o.json)
	}
	if o.logJSON {
		logger := slog.New(slog.NewJSONHandler(stderr, nil))
		logger.Info("command started", "command", o.args[0])
		defer func() { logger.Info("command finished", "command", o.args[0], "exit_code", code) }()
	}
	value, e := execute(o)
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
		return map[string]any{"version": Version, "go": runtime.Version(), "real_activation": false}, need(0)
	}
	p, e := system.Locations(o.root)
	if e != nil {
		return nil, e
	}
	switch command {
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
		return catalog.Search(q)
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
		return catalog.Find(a[0])
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
		pkg, e := state.Select(packages, canonical(a[0]))
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
			target = canonical(target)
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
		pkg, e := state.Select(packages, canonical(a[0]))
		if e != nil {
			return nil, e
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
func canonical(id string) string {
	if entry, e := catalog.Find(id); e == nil {
		return entry.Recipe.ID
	}
	return id
}
func render(w io.Writer, value any) {
	switch v := value.(type) {
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
	case planner.Plan:
		fmt.Fprintf(w, "PLAN %s\n\n%s %s@%s\nArtifact SHA256: %s\nTree SHA256: %s\nDestination: %s\nTheme: %q -> %q\nValidation: %s\nCompatibility: %s\nPrivileges: %s\nWould invoke (never executed in this build): %v\nApplicable: %t — %s\n", v.ID, v.Request.Action, v.ThemeID, v.Revision, v.ArtifactSHA256, v.TreeSHA256, v.Destination, v.BeforeTheme, v.AfterTheme, v.Validation, v.Compatibility, v.Privileges, v.Generator, v.Applicable, v.Reason)
		for _, s := range v.Add {
			fmt.Fprintln(w, "+", s)
		}
		for _, s := range v.Remove {
			fmt.Fprintln(w, "-", s)
		}
		for _, s := range v.Retained {
			fmt.Fprintln(w, "Retain:", s)
		}
		fmt.Fprintf(w, "Recovery: %s\nNo changes have been made.\n", v.Recovery)
	default:
		_ = output.Write(w, value)
	}
}
